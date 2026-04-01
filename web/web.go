// /home/krylon/go/src/github.com/blicero/newsroom/web/web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-31 14:56:45 krylon>

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
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/critic"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
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
	scrub     *scrub.Scrubber
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
	} else if srv.scrub, err = scrub.Create(); err != nil {
		srv.log.Printf("[ERROR] Cannot create Scrubber: %s\n",
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

	return srv, nil
} // func Create(addr string, nx *nexus.Nexus) (*Server, error)

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
	}

	for _, item := range data.Items {
		if err = srv.scrub.Scrub(item); err != nil {
			srv.log.Printf("[ERROR] Failed to scrub Item %d (%s): %s\n",
				item.ID,
				item.Title,
				err.Error())
		}

		if item.IsRated() {
			continue
		} else if item.GuessedRating, err = srv.cls.Classify(item); err != nil {
			srv.log.Printf("[ERROR] Failed to classify Item %d (%s): %s\n",
				item.ID,
				item.Title,
				err.Error())
			item.GuessedRating = rating.Unrated
		}
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

func (srv *Server) handleRetrain(w http.ResponseWriter, r *http.Request) {
} // func (srv *Server) handleRetrain(w http.ResponseWriter, r *http.Request)

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

	srv.log.Printf("[TRACE] Delivering static file %s to client %s\n",
		filename,
		request.RemoteAddr)

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
