// /home/krylon/go/src/github.com/blicero/newsroom/web/web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-21 15:22:16 krylon>

package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/blicero/newsroom/blacklist"
	"github.com/blicero/newsroom/classify"
	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/critic"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
	"github.com/blicero/newsroom/scrub"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
)

const (
	cacheControl = "max-age=3600, public"
	noCache      = "no-store, max-age=0"
	tmplFolder   = "assets/templates"
)

//go:embed assets
var assets embed.FS

// Server provides a web-based UI
type Server struct {
	addr      string
	log       *log.Logger
	pool      *database.Pool // nolint: unused
	lock      sync.RWMutex   // nolint: unused
	active    atomic.Bool
	cls       *critic.Critic
	adv       *classify.Advisor
	scrub     *scrub.Scrubber
	bl        *blacklist.Blacklist
	router    *mux.Router
	tmpl      *template.Template
	web       http.Server
	mimeTypes map[string]string
}

// Create returns a new web Server.
func Create(addr string) (*Server, error) {
	var (
		err error
		msg string
		srv = &Server{
			addr: addr,
			mimeTypes: map[string]string{
				".css":  "text/css",
				".map":  "application/json",
				".js":   "text/javascript",
				".png":  "image/png",
				".jpg":  "image/jpeg",
				".jpeg": "image/jpeg",
				".webp": "image/webp",
				".gif":  "image/gif",
				".json": "application/json",
				".html": "text/html",
			},
		}
	)

	if srv.log, err = common.GetLogger(logdomain.Web); err != nil {
		return nil, err
	} else if srv.pool, err = database.NewPool(4); err != nil {
		srv.log.Printf("[CRITICAL] Cannot open database pool: %s\n",
			err.Error())
		return nil, err
	} else if srv.cls, err = critic.New(); err != nil {
		srv.log.Printf("[CRITICAL] Cannot create Classifier: %s\n",
			err.Error())
		return nil, err
	} else if srv.adv, err = classify.New(); err != nil {
		srv.log.Printf("[CRITICAL] Cannot create Tag Advisor: %s\n",
			err.Error())
		return nil, err
	} else if srv.scrub, err = scrub.Create(); err != nil {
		srv.log.Printf("[ERROR] Cannot create Scrubber: %s\n",
			err.Error())
		return nil, err
	} else if srv.bl, err = blacklist.New(); err != nil {
		srv.log.Printf("[ERROR] Failed to create Blacklist: %s\n",
			err.Error())
		return nil, err
	}

	var templates []fs.DirEntry
	var tmplRe = regexp.MustCompile("[.]tmpl$")

	if templates, err = assets.ReadDir(tmplFolder); err != nil {
		srv.log.Printf("[ERROR] Cannot read embedded templates: %s\n",
			err.Error())
		return nil, err
	}

	srv.tmpl = template.New("").Funcs(funcmap)
	for _, entry := range templates {
		var (
			content []byte
			path    = filepath.Join(tmplFolder, entry.Name())
		)

		if !tmplRe.MatchString(entry.Name()) {
			continue
		} else if content, err = assets.ReadFile(path); err != nil {
			msg = fmt.Sprintf("Cannot read embedded file %s: %s",
				path,
				err.Error())
			srv.log.Printf("[CRITICAL] %s\n", msg)
			return nil, errors.New(msg)
		} else if srv.tmpl, err = srv.tmpl.Parse(string(content)); err != nil {
			msg = fmt.Sprintf("Could not parse template %s: %s",
				entry.Name(),
				err.Error())
			srv.log.Println("[CRITICAL] " + msg)
			return nil, errors.New(msg)
		} else if common.Debug {
			srv.log.Printf("[TRACE] Template \"%s\" was parsed successfully.\n",
				entry.Name())
		}
	}

	srv.router = mux.NewRouter()
	srv.web.Addr = addr
	srv.web.ErrorLog = srv.log
	srv.web.Handler = srv.router

	// Register URL handlers
	srv.router.NotFoundHandler = http.HandlerFunc(srv.handleNotFound)
	srv.router.HandleFunc("/favicon.ico", srv.handleFavIco)
	srv.router.HandleFunc("/static/{file}", srv.handleStaticFile)
	srv.router.HandleFunc("/{index:(?i:index|main|start)$}", srv.handleMain)
	srv.router.HandleFunc("/news/{pageno:(?:\\d+)}/{cnt:(?:\\d+)$}", srv.handleNews)
	srv.router.HandleFunc("/feed/all", srv.handleSubscriptions)
	srv.router.HandleFunc("/tag/all", srv.handleTagsView)
	srv.router.HandleFunc("/blacklist", srv.handleBlacklistView)
	srv.router.HandleFunc("/retrain_classifier", srv.handleRetrain)
	srv.router.HandleFunc("/search", srv.handleSearchForm)

	// AJAX Handlers
	srv.router.HandleFunc(
		"/ajax/beacon",
		srv.handleBeacon)
	srv.router.HandleFunc(
		"/ajax/item_rate/{id:(?:\\d+)}/{rating:(?:\\d+)$}",
		srv.handleAjaxRateItem,
	)
	srv.router.HandleFunc(
		"/ajax/item_unrate/{id:(?:\\d+)$}",
		srv.handleAjaxUnrateItem,
	)
	srv.router.HandleFunc(
		"/ajax/subscribe",
		srv.handleAjaxSubscribe,
	)
	srv.router.HandleFunc(
		"/ajax/feed/toggle_active/{id:(?:\\d+)$}",
		srv.handleAjaxFeedToggleActive,
	)
	srv.router.HandleFunc(
		"/ajax/tag/submit",
		srv.handleAjaxTagSubmit,
	)
	srv.router.HandleFunc(
		"/ajax/tag_link/create",
		srv.handleAjaxTagLinkCreate,
	)
	srv.router.HandleFunc(
		"/ajax/tag_link/delete",
		srv.handleAjaxTagLinkRemove,
	)
	srv.router.HandleFunc(
		"/ajax/blacklist/add",
		srv.handleAjaxBlacklistAdd,
	)
	srv.router.HandleFunc(
		"/ajax/blacklist/remove",
		srv.handleAjaxBlacklistRemove,
	)

	return srv, nil
} // func Create(addr string) (*Server, error)

// IsActive returns the Server's active flag.
func (srv *Server) IsActive() bool {
	return srv.active.Load()
} // func (srv *Server) IsActive() bool

// Stop clears the Server's active flag.
func (srv *Server) Stop() {
	srv.active.Store(false)
} // func (srv *Server) Stop()

// Run executes the Server's loop, waiting for new connections and starting
// goroutines to handle them.
func (srv *Server) Run() {
	var err error

	defer srv.log.Println("[INFO] Web server is shutting down")

	srv.active.Store(true)
	defer srv.active.Store(false)

	srv.log.Printf("[INFO] Web frontend is going online at %s\n", srv.addr)
	http.Handle("/", srv.router)

	if err = srv.web.ListenAndServe(); err != nil {
		if err.Error() != "http: Server closed" {
			srv.log.Printf("[ERROR] ListenAndServe returned an error: %s\n",
				err.Error())
		} else {
			srv.log.Println("[INFO] HTTP Server has shut down.")
		}
	}
} // func (srv *Server) Run()

//////////////////////////////////////////////////////////////////////////////
/// Handle requests //////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////

func (srv *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	srv.log.Printf("[ERROR] 404 - %s\n", r.RequestURI)

	srv.sendErrorMessage(
		w,
		fmt.Sprintf(
			"No Handler was found for %s",
			r.RequestURI))
} // func (srv *Server) handleNotFound(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleMain(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	const tmplName = "main"

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataIndex{
			tmplDataBase: tmplDataBase{
				Title: "Main",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Feeds, err = db.FeedGetAll(); err != nil {
		msg = fmt.Sprintf("Failed to load Feeds from Database: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleMain(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleNews(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	const tmplName = "news"

	var (
		err   error
		msg   string
		db    *database.Database
		tmpl  *template.Template
		feeds []*model.Feed
		vars  map[string]string
		data  = tmplDataNews{
			tmplDataBase: tmplDataBase{
				Title: "News",
				Debug: common.Debug,
				URL:   r.URL.String(),
			},
		}
	)

	vars = mux.Vars(r)

	if data.PageNo, err = strconv.ParseInt(vars["pageno"], 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse offset %q: %s",
			vars["offset"],
			err.Error())
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Count, err = strconv.ParseInt(vars["cnt"], 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse item count %q: %s",
			vars["cnt"],
			err.Error())
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Could not find template %q", tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if feeds, err = db.FeedGetAll(); err != nil {
		msg = fmt.Sprintf("Failed to load Feeds from Database: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Items, err = db.ItemGetAll(data.Count, data.Count*data.PageNo); err != nil {
		msg = fmt.Sprintf("Failed to load %d recent Items: %s",
			data.Count,
			err.Error())
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.TotalCount, err = db.ItemCount(); err != nil {
		msg = fmt.Sprintf("Failed to query total count of Items: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Tags, err = db.TagGetSorted(); err != nil {
		msg = fmt.Sprintf("Failed to load all Tags: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.TagAdvice = make(map[int64]classify.SuggList, len(data.Items))
	data.ItemTags = make(map[int64]map[int64]bool, len(data.Items))
	data.TagMap = make(map[int64]*model.Tag, len(data.Tags))
	for _, tag := range data.Tags {
		data.TagMap[tag.ID] = tag
	}

	var blHit int

	defer func() {
		if blHit > 0 {
			srv.bl.Save()
		}
	}()

	data.Items = slices.DeleteFunc(data.Items, func(item *model.Item) bool {
		if srv.bl.Match(item) {
			srv.bl.Sort()
			blHit++
			return true
		}
		return false
	})

	for _, item := range data.Items {
		if err = srv.scrub.Scrub(item); err != nil {
			srv.log.Printf("[ERROR] Failed to scrub Item %d (%s): %s\n",
				item.ID,
				item.Title,
				err.Error())
		}

		if !item.IsRated() {
			if item.GuessedRating, err = srv.cls.Classify(item); err != nil {
				srv.log.Printf("[ERROR] Failed to classify Item %d (%s): %s\n",
					item.ID,
					item.Title,
					err.Error())
				item.GuessedRating = rating.Unrated
			}
		}

		var itemTags []*model.Tag

		if itemTags, err = db.TagLinkGetByItem(item); err != nil {
			msg = fmt.Sprintf("Failed to load Tags for Item %q (%d): %s",
				item.Title,
				item.ID,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			srv.sendErrorMessage(w, msg)
			return
		} else if data.TagAdvice[item.ID], err = srv.adv.Score(item); err != nil {
			msg = fmt.Sprintf("Failed to calculate Tag Advice for Item %q (%d): %s",
				item.Title,
				item.ID,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			srv.sendErrorMessage(w, msg)
			return
		}

		// data.ItemTags[item.ID] = make(map[int64]bool, len(itemTags))
		var itags = make(map[int64]bool, len(itemTags))
		for _, tag := range itemTags {
			itags[tag.ID] = true
		}

		data.ItemTags[item.ID] = itags
	}

	data.MaxPage = data.TotalCount / data.Count
	data.Feeds = make(map[int64]*model.Feed, len(feeds))

	for _, feed := range feeds {
		data.Feeds[feed.ID] = feed
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleNews(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	const tmplName = "feeds"

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataIndex{
			tmplDataBase: tmplDataBase{
				Title: "Manage Subscriptions",
				Debug: common.Debug,
				URL:   r.RequestURI,
			},
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Couldn't find template %s",
			tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Feeds, err = db.FeedGetAll(); err != nil {
		msg = fmt.Sprintf("Failed to load Feeds from Database: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleTagsView(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	const tmplName = "tags"

	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataTags{
			tmplDataBase: tmplDataBase{
				Title: "Manage Tags",
				Debug: common.Debug,
				URL:   r.RequestURI,
			},
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Couldn't find template %s",
			tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Tags, err = db.TagGetSorted(); err != nil {
		msg = fmt.Sprintf("Failed to load Feeds from Database: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleTagsView(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleRetrain(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var err error

	if err = srv.cls.Reset(); err != nil {
		srv.log.Printf("[ERROR] Failed to retrain Classifier: %s\n",
			err.Error())
	} else if err = srv.adv.Reset(); err != nil {
		srv.log.Printf("[ERROR] Failed to retrain Tag Advisor: %s\n",
			err.Error())
	}

	http.Redirect(w, r, r.Referer(), 307)
} // func (srv *Server) handleRetrain(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleBlacklistView(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)

	const tmplName = "blacklist"
	var (
		err  error
		msg  string
		tmpl *template.Template
		data = tmplDataBlacklist{
			tmplDataBase: tmplDataBase{
				Title: "Manage Blacklist",
				Debug: common.Debug,
				URL:   r.RequestURI,
			},
		}
	)

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Couldn't find template %s",
			tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.Patterns = srv.bl.Patterns()

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleBlacklistView(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleSearchForm(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	const tmplName = "search"
	var (
		err  error
		msg  string
		db   *database.Database
		tmpl *template.Template
		data = tmplDataSearch{
			tmplDataNews: tmplDataNews{
				tmplDataBase: tmplDataBase{
					Title: "Search",
					Debug: common.Debug,
					URL:   r.RequestURI,
				},
			},
		}
	)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if data.Tags, err = db.TagGetSorted(); err != nil {
		msg = fmt.Sprintf("Failed to load Tags from database: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.TagMap = make(map[int64]*model.Tag, len(data.Tags))

	for _, tag := range data.Tags {
		data.TagMap[tag.ID] = tag
	}

	if strings.ToLower(r.Method) == "post" {
		// We should process the search query.
		data.Messages = make([]string, 1)
		data.Messages[0] = "Actually PERFORMING the search is not there, yet."
		srv.performSearch(db, w, r)
		return
	} else if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Couldn't find template %s",
			tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) handleSearchForm(w http.ResponseWriter, r *http.Request)

// Since handleSearchForm is long enough as it is, I delegate the actual searching
// to this method.
func (srv *Server) performSearch(db *database.Database, w http.ResponseWriter, r *http.Request) {
	const (
		tagPrefix = "tag_"
		tmplName  = "search"
	)

	var (
		err              error
		msg, query       string
		tmpl             *template.Template
		tags             = make(map[int64]bool)
		dateP, tagP      bool
		dateFrom, dateTo time.Time
		feeds            []*model.Feed
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Failed to parse form data: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	msg = spew.Sdump(r.PostForm)
	srv.log.Printf("[DEBUG] We got a search query:\n\n%s\n\n",
		msg)

	if r.FormValue("tag_p") == "on" {
		tagP = true
		for key := range r.PostForm {
			var id int64
			if !strings.HasPrefix(key, tagPrefix) {
				continue
			} else if id, err = strconv.ParseInt(key[4:], 10, 64); err != nil {
				if key == "tag_p" {
					continue
				}

				msg = fmt.Sprintf("Failed to parse Tag ID %s: %s",
					key,
					err.Error())
				srv.log.Printf("[ERROR] %s\n", msg)
				srv.sendErrorMessage(w, msg)
				return
			}
			tags[id] = true
		}
	}

	if r.FormValue("date_p") == "on" {
		dateP = true
		var dstr = r.FormValue("date_begin")
		if dateFrom, err = time.Parse(common.TimestampFormatDate, dstr); err != nil {
			msg = fmt.Sprintf("Failed to parse begin date %q: %s",
				dstr,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			srv.sendErrorMessage(w, msg)
			return
		}

		dstr = r.FormValue("date_to")
		if dateTo, err = time.Parse(common.TimestampFormatDate, dstr); err != nil {
			msg = fmt.Sprintf("Failed to parse end date %q: %s",
				dstr,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			srv.sendErrorMessage(w, msg)
			return
		}
	}

	query = r.FormValue("query")

	if !strings.HasPrefix(query, "%") && !strings.HasSuffix(query, "%") {
		query = "%" + query + "%"
	}

	var parm = database.SearchParms{
		DateP: dateP,
		TagP:  tagP,
		Query: query,
	}

	if dateP {
		parm.DateRange[0] = dateFrom
		parm.DateRange[1] = dateTo
	}

	if tagP {
		parm.Tags = tags
	}

	var data = tmplDataSearch{
		tmplDataNews: tmplDataNews{
			tmplDataBase: tmplDataBase{
				Title: "Search",
				Debug: common.Debug,
				URL:   r.RequestURI,
			},
		},
	}

	if data.Items, err = db.Search(&parm); err != nil {
		msg = fmt.Sprintf("Search failed: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if data.Tags, err = db.TagGetSorted(); err != nil {
		msg = fmt.Sprintf("Failed to load Tags: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if feeds, err = db.FeedGetAll(); err != nil {
		msg = fmt.Sprintf("Failed to load Feeds: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	data.Feeds = make(map[int64]*model.Feed, len(feeds))

	for _, feed := range feeds {
		data.Feeds[feed.ID] = feed
	}

	data.TagMap = make(map[int64]*model.Tag, len(data.Tags))

	for _, tag := range data.Tags {
		data.TagMap[tag.ID] = tag
	}

	data.ItemTags = make(map[int64]map[int64]bool, len(data.Items))

	data.TagAdvice = make(map[int64]classify.SuggList, len(data.Items))

	for _, item := range data.Items {
		var (
			taglist []*model.Tag
			tmap    map[int64]bool
		)

		if taglist, err = db.TagLinkGetByItem(item); err != nil {
			msg = fmt.Sprintf("Failed to load Tags for Item %q (%d): %s",
				item.Title,
				item.ID,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			srv.sendErrorMessage(w, msg)
			return
		} else if data.TagAdvice[item.ID], err = srv.adv.Score(item); err != nil {
			msg = fmt.Sprintf("Failed to calculate Tag Advice for Item %q (%d): %s",
				item.Title,
				item.ID,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			srv.sendErrorMessage(w, msg)
			return
		}

		tmap = make(map[int64]bool, len(taglist))

		for _, tag := range taglist {
			tmap[tag.ID] = true
		}

		data.ItemTags[item.ID] = tmap
	}

	if tmpl = srv.tmpl.Lookup(tmplName); tmpl == nil {
		msg = fmt.Sprintf("Couldn't find template %s",
			tmplName)
		srv.log.Println("[CRITICAL] " + msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.Header().Set("Cache-Control", noCache)
	if err = tmpl.Execute(w, &data); err != nil {
		msg = fmt.Sprintf("Error rendering template %q: %s",
			tmplName,
			err.Error())
		srv.sendErrorMessage(w, msg)
	}
} // func (srv *Server) performSearch(db *database.Database, w http.ResponseWriter, r *http.Request)

//////////////////////////////////////////////////////////////////////////////
/// Handle AJAX //////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////

func (srv *Server) handleAjaxRateItem(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err              error
		vars             map[string]string
		msg, idstr, rstr string
		id, rint         int64
		score            rating.Rating
		item             *model.Item
		db               *database.Database
		buf              []byte
		data             = ajaxResponseRateItem{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	vars = mux.Vars(r)

	idstr = vars["id"]
	rstr = vars["rating"]

	if id, err = strconv.ParseInt(idstr, 10, 64); err != nil {
		msg = fmt.Sprintf("Failed to parse Item ID %q: %s",
			idstr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if rint, err = strconv.ParseInt(rstr, 10, 64); err != nil {
		msg = fmt.Sprintf("Failed to parse Rating %q: %s",
			rstr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	score = rating.Rating(rint)

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if item, err = db.ItemGetByID(id); err != nil {
		msg = fmt.Sprintf("Failed to look up Item #%d: %s",
			id,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if item == nil {
		msg = fmt.Sprintf("Item #%d does not exist", id)
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if err = db.ItemSetRating(item, rating.Rating(rint)); err != nil {
		msg = fmt.Sprintf("Failed to rate Item %d (%s) as %s: %s",
			item.ID,
			item.Title,
			score,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if err = srv.cls.Learn(item); err != nil {
		msg = fmt.Sprintf("Failed to teach the Classifier about Item %d (%s) as %s: %s",
			item.ID,
			item.Title,
			score,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	data.Status = true
	data.Message = fmt.Sprintf("Item %d was rated as %s",
		id,
		rating.Rating(rint))

	if buf, err = json.Marshal(&data); err != nil {
		var msg = fmt.Sprintf("Failed to serialize payload for AJAX response: %s",
			err.Error())
		srv.log.Printf("[CANTHAPPEN] %s\n", msg)
		buf = errJSON(msg)
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleRateItem(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxUnrateItem(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err        error
		vars       map[string]string
		msg, idstr string
		id         int64
		item       *model.Item
		db         *database.Database
		buf        []byte
		data       = ajaxResponseRateItem{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	vars = mux.Vars(r)
	idstr = vars["id"]

	if id, err = strconv.ParseInt(idstr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse ID %q: %s",
			idstr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if item, err = db.ItemGetByID(id); err != nil {
		msg = fmt.Sprintf("Failed to load Item #%d: %s",
			id,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if item == nil {
		msg = fmt.Sprintf("Item #%d was not found in Database", id)
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if err = db.ItemSetRating(item, rating.Unrated); err != nil {
		msg = fmt.Sprintf("Failed to unrate Item %d: %s",
			id,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if err = srv.cls.Unlearn(item); err != nil {
		msg = fmt.Sprintf("Failed to make the Classifier forget about Item %d (%s): %s",
			item.ID,
			item.Title,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if _, err = srv.cls.Classify(item); err != nil {
		msg = fmt.Sprintf("Failed to classify Item %d (%s) after unlearning it: %s",
			item.ID,
			item.Title,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	data.Status = true
	data.Message = "Success!"
	data.Content = fmt.Sprintf(`
        <small><i> %s </i></small>
        <button type="button"
                class="btn btn-primary btn-sm"
                onclick="rate_item(%d, 2);">
            Interesting
        </button>
        <button type="button"
                class="btn btn-secondary btn-sm"
                onclick="rate_item(%d, 1);">
            Boring
        </button>
`,
		item.EffectiveRating(),
		id,
		id)

	if buf, err = json.Marshal(&data); err != nil {
		msg = fmt.Sprintf("Failed to convert data to JSON: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxUnrateItem(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxSubscribe(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err         error
		msg         string
		db          *database.Database
		intervalStr string
		interval    int64
		feed        model.Feed
		buf         []byte
		data        = ajaxData{
			Timestamp: time.Now(),
		}
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Cannot parse form data: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if feed.URL, err = url.Parse(r.FormValue("url")); err != nil {
		msg = fmt.Sprintf("Failed to parse Feed URL %q: %s",
			r.FormValue("url"),
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if feed.Homepage, err = url.Parse(r.FormValue("homepage")); err != nil {
		msg = fmt.Sprintf("Failed to parse Feed Homepage %q: %s",
			r.FormValue("homepage"),
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	intervalStr = r.FormValue("interval")
	if interval, err = strconv.ParseInt(intervalStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse interval %q: %s",
			intervalStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	feed.Name = r.FormValue("name")
	feed.RefreshInterval = time.Second * time.Duration(interval)
	feed.Language = r.FormValue("lang")

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if err = db.FeedAdd(&feed); err != nil {
		msg = fmt.Sprintf("Adding Feed %s to Database failed: %s",
			feed.Name,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	data.Status = true
	data.Message = fmt.Sprintf("Subscription to %s was added successfully",
		feed.Name)
	if buf, err = json.Marshal(&data); err != nil {
		msg = fmt.Sprintf("Failed to serialize response: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxSubscribe(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxFeedToggleActive(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err        error
		vars       map[string]string
		msg, idstr string
		id         int64
		db         *database.Database
		feed       *model.Feed
		buf        []byte
		data       = ajaxData{
			Timestamp: time.Now(),
		}
	)

	vars = mux.Vars(r)
	idstr = vars["id"]

	if id, err = strconv.ParseInt(idstr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse ID %q: %s",
			idstr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if feed, err = db.FeedGetByID(id); err != nil {
		msg = fmt.Sprintf("Lookup of Feed %d failed: %s",
			id,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if feed == nil {
		msg = fmt.Sprintf("Feed %d does not exist in Database",
			id)
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if err = db.FeedSetActive(feed, !feed.Active); err != nil {
		msg = fmt.Sprintf("Error toggling Active flag of Feed %s (%d): %s",
			feed.Name,
			feed.ID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	data.Status = true
	data.Message = fmt.Sprintf("Active flag of Feed %s (%d) has been toggled successfully",
		feed.Name,
		feed.ID)

	if buf, err = json.Marshal(&data); err != nil {
		msg = fmt.Sprintf("Failed to serialize response: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxFeedToggleActive(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxTagSubmit(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err                error
		msg, idstr, parstr string
		db                 *database.Database
		buf                []byte
		tag                model.Tag
		dtag               *model.Tag
		res                = ajaxResponseTagSubmit{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Cannot parse form data: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	idstr = r.FormValue("id")
	parstr = r.FormValue("parent")
	tag.Name = r.FormValue("name")

	if idstr != "" {
		if tag.ID, err = strconv.ParseInt(idstr, 10, 64); err != nil {
			msg = fmt.Sprintf("Cannot parse Tag ID %q: %s",
				idstr,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			buf = errJSON(msg)
			goto SEND
		}
	}

	if parstr != "" {
		if tag.ParentID, err = strconv.ParseInt(parstr, 10, 64); err != nil {
			msg = fmt.Sprintf("Cannot parse Parent ID %q: %s",
				parstr,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			buf = errJSON(msg)
			goto SEND
		}
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if tag.ID == 0 {
		if err = db.TagAdd(&tag); err != nil {
			msg = fmt.Sprintf("Failed to add Tag %q to Database: %s",
				tag.Name,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			buf = errJSON(msg)
			goto SEND
		}

		res.Status = true
		res.Message = fmt.Sprintf("Tag %s was added successfully",
			tag.Name)
		goto JSON

	} else if dtag, err = db.TagGetByID(tag.ID); err != nil {
		msg = fmt.Sprintf("Failed to load Tag #%d from Database: %s",
			tag.ID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if dtag == nil {
		msg = fmt.Sprintf("Tag #%d was not found in Database!",
			tag.ID)
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	if tag.ParentID != dtag.ParentID {
		if err = db.TagSetParent(dtag, tag.ParentID); err != nil {
			msg = fmt.Sprintf("Failed to set Parent of Tag %s (%d) to %d: %s",
				dtag.Name,
				dtag.ID,
				tag.ParentID,
				err.Error())
			srv.log.Printf("[ERROR] %s\n", msg)
			buf = errJSON(msg)
			goto SEND
		}
	}

	if tag.Name != dtag.Name {
		msg = "Renaming Tags is not allowed"
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	res.Status = true
	res.Message = "Success"

JSON:
	if buf, err = json.Marshal(&res); err != nil {
		msg = fmt.Sprintf("Failed to serialize Response to JSON: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxTagSubmit(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxTagLinkCreate(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err                  error
		msg, itemStr, tagStr string
		db                   *database.Database
		buf                  []byte
		lnk                  model.TagLink
		res                  = ajaxResponseTagLinkCreate{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Cannot parse form data: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	itemStr = r.FormValue("item_id")
	tagStr = r.FormValue("tag_id")

	if lnk.ItemID, err = strconv.ParseInt(itemStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse Item ID %q: %s",
			itemStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if lnk.TagID, err = strconv.ParseInt(tagStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse Tag ID %q: %s",
			tagStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if err = db.TagLinkAdd(&lnk); err != nil {
		msg = fmt.Sprintf("Failed to create TagLink(%d -> %d): %s",
			lnk.TagID,
			lnk.ItemID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if res.Tag, err = db.TagGetByID(lnk.TagID); err != nil {
		msg = fmt.Sprintf("Failed load Tag #%d: %s",
			lnk.TagID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	res.Status = true
	res.Message = "Success"

	if buf, err = json.Marshal(&res); err != nil {
		msg = fmt.Sprintf("Failed to serialize response: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxTagLinkCreate(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxTagLinkRemove(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err                  error
		msg, itemStr, tagStr string
		db                   *database.Database
		buf                  []byte
		lnk                  model.TagLink
		res                  = ajaxResponseTagLinkCreate{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Cannot parse form data: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	itemStr = r.FormValue("item_id")
	tagStr = r.FormValue("tag_id")

	if lnk.ItemID, err = strconv.ParseInt(itemStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse Item ID %q: %s",
			itemStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	} else if lnk.TagID, err = strconv.ParseInt(tagStr, 10, 64); err != nil {
		msg = fmt.Sprintf("Cannot parse Tag ID %q: %s",
			tagStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	db = srv.pool.Get()
	defer srv.pool.Put(db)

	if err = db.TagLinkDelete(lnk.TagID, lnk.ItemID); err != nil {
		msg = fmt.Sprintf("Failed to create TagLink(%d -> %d): %s",
			lnk.TagID,
			lnk.ItemID,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	res.Status = true
	res.Message = "Success"

	if buf, err = json.Marshal(&res); err != nil {
		msg = fmt.Sprintf("Failed to serialize response: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxTagLinkRemove(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxBlacklistAdd(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err      error
		msg, pat string
		buf      []byte
		res      = ajaxData{
			Timestamp: time.Now(),
		}
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Cannot parse form data: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	pat = r.FormValue("pattern")

	if err = srv.bl.Add(pat); err != nil {
		msg = fmt.Sprintf("Failed to add pattern %q to Blacklist: %s",
			pat,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	res.Status = true
	res.Message = "Success"

	if buf, err = json.Marshal(&res); err != nil {
		msg = fmt.Sprintf("Failed to serialize response: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxBlacklistAdd(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleAjaxBlacklistRemove(w http.ResponseWriter, r *http.Request) {
	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	var (
		err      error
		msg, pat string
		buf      []byte
		res      = ajaxData{
			Timestamp: time.Now(),
		}
	)

	if err = r.ParseForm(); err != nil {
		msg = fmt.Sprintf("Cannot parse form data: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	pat = r.FormValue("pattern")

	if err = srv.bl.Remove(pat); err != nil {
		msg = fmt.Sprintf("Failed to add pattern %q to Blacklist: %s",
			pat,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

	res.Status = true
	res.Message = "Success"

	if buf, err = json.Marshal(&res); err != nil {
		msg = fmt.Sprintf("Failed to serialize response: %s",
			err.Error())
		srv.log.Printf("[ERROR] %s\n", msg)
		buf = errJSON(msg)
		goto SEND
	}

SEND:
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *Server) handleAjaxBlacklistRemove(w http.ResponseWriter, r *http.Request)

func (srv *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		buf  []byte
		data = ajaxBeaconData{
			ajaxData: ajaxData{
				Status:    true,
				Timestamp: time.Now(),
				Message:   common.AppName + " " + common.Version,
			},
			Hostname: hostname(),
		}
	)

	if buf, err = json.Marshal(&data); err != nil {
		var msg = fmt.Sprintf("Failed to serialize payload for AJAX response: %s",
			err.Error())
		srv.log.Printf("[CANTHAPPEN] %s\n", msg)
		buf = errJSON(msg)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", noCache)
	w.WriteHeader(200)
	w.Write(buf) // nolint: errcheck,gosec
} // func (srv *WebFrontend) handleBeacon(w http.ResponseWriter, r *http.Request)

//////////////////////////////////////////////////////////////////////////////
/// Handle static assets /////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////

func (srv *Server) handleFavIco(w http.ResponseWriter, request *http.Request) {
	srv.log.Printf("[TRACE] Handle request for %s\n",
		request.URL.EscapedPath())

	const (
		filename = "assets/static/favicon.ico"
		mimeType = "image/vnd.microsoft.icon"
	)

	w.Header().Set("Content-Type", mimeType)

	if !common.Debug {
		w.Header().Set("Cache-Control", cacheControl)
	} else {
		w.Header().Set("Cache-Control", noCache)
	}

	var (
		err error
		fh  fs.File
	)

	if fh, err = assets.Open(filename); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", filename)
		srv.sendErrorMessage(w, msg)
	} else {
		defer fh.Close() // nolint: errcheck
		w.WriteHeader(200)
		io.Copy(w, fh) // nolint: errcheck
	}
} // func (srv *Server) handleFavIco(w http.ResponseWriter, request *http.Request)

func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request) {
	// srv.log.Printf("[TRACE] Handle request for %s\n",
	// 	request.URL.EscapedPath())

	// Since we controll what static files the server has available, we
	// can easily map MIME type to slice. Soon.

	vars := mux.Vars(request)
	filename := vars["file"]
	path := filepath.Join("assets", "static", filename)

	var mimeType string

	// srv.log.Printf("[TRACE] Delivering static file %s to client %s\n",
	// 	filename,
	// 	request.RemoteAddr)

	var match []string

	if match = common.SuffixPattern.FindStringSubmatch(filename); match == nil {
		mimeType = "text/plain"
	} else if mime, ok := srv.mimeTypes[match[1]]; ok {
		mimeType = mime
	} else {
		srv.log.Printf("[ERROR] Did not find MIME type for %s\n", filename)
	}

	w.Header().Set("Content-Type", mimeType)

	if common.Debug {
		w.Header().Set("Cache-Control", noCache)
	} else {
		w.Header().Set("Cache-Control", cacheControl)
	}

	var (
		err error
		fh  fs.File
	)

	if fh, err = assets.Open(path); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", path)
		srv.sendErrorMessage(w, msg)
	} else {
		defer fh.Close() // nolint: errcheck
		w.WriteHeader(200)
		io.Copy(w, fh) // nolint: errcheck
	}
} // func (srv *Server) handleStaticFile(w http.ResponseWriter, request *http.Request)

func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string) {
	html := `
<!DOCTYPE html>
<html>
  <head>
    <title>Internal Error</title>
  </head>
  <body>
    <h1>Internal Error</h1>
    <hr />
    We are sorry to inform you an internal application error has occured:<br />
    %s
    <p>
    Back to <a href="/index">Homepage</a>
    <hr />
    &copy; 2026 <a href="mailto:krylon@gmx.net">Benjamin Walkenhorst</a>
  </body>
</html>
`

	w.Header().Set("Cache-Control", noCache)
	srv.log.Printf("[ERROR] %s\n", msg)

	output := fmt.Sprintf(html, msg)
	w.WriteHeader(500)
	_, _ = w.Write([]byte(output)) // nolint: gosec
} // func (srv *Server) sendErrorMessage(w http.ResponseWriter, msg string)
