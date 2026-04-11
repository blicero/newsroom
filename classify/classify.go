// /home/krylon/go/src/github.com/blicero/newsroom/classify/classify.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-11 15:38:40 krylon>

// Package classify suggests Tags that are likely to be a match for news Items.
package classify

import (
	"cmp"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/blicero/newsroom/cache"
	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/shield"
)

// If a query returns an error and the error text is matched by this regex, we
// consider the error as transient and try again after a short delay.
var retryPat = regexp.MustCompile("(?i)resource temporarily unavailable")

// worthARetry returns true if an error returned from the database
// is matched by the retryPat regex.
func worthARetry(e error) bool {
	return retryPat.MatchString(e.Error())
} // func worthARetry(e error) bool

// retryDelay is the amount of time we wait before we repeat a database
// operation that failed due to a transient error.
const (
	retryDelay = 25 * time.Millisecond
	maxWait    = 10
)

func waitForRetry() {
	time.Sleep(retryDelay)
} // func waitForRetry()

// REMINDER: To avoid problems when Tags are renamed, I should avoid using their
//           names as class names. I have to use a string because of the API
//           offered by Shield, but I could use e.g. the stringified IDs.

// Suggestion is the ID of a Tag and the score calculated by the Advisor.
type Suggestion struct {
	TagID int64
	Match float64
}

// SuggList is a list of Suggestions.
type SuggList []Suggestion

var languages = []string{"en", "de"}

type score struct {
	ID   int64
	Tags map[string]float64
}

// Advisor suggests Tags for Items, based on existing tagging.
type Advisor struct {
	log      *log.Logger
	advisors map[string]shield.Shield
	db       *database.Database
	lock     sync.RWMutex
	acache   *cache.Cache[score]
	lngMap   map[int64]string
}

// New creates a new Advisor.
func New() (*Advisor, error) {
	var (
		err   error
		feeds []*model.Feed
		ad    = &Advisor{
			advisors: make(map[string]shield.Shield, len(languages)),
		}
	)

	if ad.log, err = common.GetLogger(logdomain.Classifier); err != nil {
		return nil, err
	} else if ad.db, err = database.Open(common.DbPath); err != nil {
		ad.log.Printf("[CRITICAL] Cannot open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	} else if feeds, err = ad.db.FeedGetAll(); err != nil {
		ad.log.Printf("[ERROR] Failed to load Feeds from Database: %s\n",
			err.Error())
		return nil, err
	}

	ad.lngMap = make(map[int64]string, len(feeds))

	for _, feed := range feeds {
		ad.lngMap[feed.ID] = feed.Language
	}

	feeds = nil

	for _, lng := range languages {
		var (
			tok       shield.Tokenizer
			storePath = filepath.Join(common.CachePath, fmt.Sprintf("advisor_model_%s", lng))
			store     = shield.NewLevelDBStore(storePath)
		)

		switch lng {
		case "en":
			tok = shield.NewEnglishTokenizer()
		case "de":
			tok = shield.NewGermanTokenizer()
		default:
			ad.log.Printf("[CANTHAPPEN] Unsupported language %s\n",
				lng)
			return nil, fmt.Errorf("unsupported language %s", lng)
		}

		ad.advisors[lng] = shield.New(tok, store)
	}

	if ad.acache, err = cache.New[score]("advisor"); err != nil {
		ad.log.Printf("[CRITICAL] Cannot open/create cache for advice: %s\n",
			err.Error())
		return nil, err
	}

	return ad, nil
} // func New() (*Advisor, error)

// Reset discards the Advisor's training state and trains it from scratch on the
// tagged Items in the Database.
func (ad *Advisor) Reset() error {
	var (
		feeds []*model.Feed
		err   error
	)

	ad.log.Println("[TRACE] Resetting/Retraining Advisor")

	ad.lock.Lock()
	defer ad.lock.Unlock()

	if feeds, err = ad.db.FeedGetAll(); err != nil {
		ad.log.Printf("[ERROR] Failed to load Feeds from Database: %s\n",
			err.Error())
		return err
	}

	clear(ad.lngMap)

	for _, feed := range feeds {
		ad.lngMap[feed.ID] = feed.Language
	}

	feeds = nil

	for lng, s := range ad.advisors {
		ad.log.Printf("[TRACE] Reset Shield for %s\n", lng)
		if err = s.Reset(); err != nil {
			ad.log.Printf("[CRITICAL] Failed to reset Shield for %s: %s\n",
				lng, err.Error())
			return err
		}
	}

	if err = ad.acache.Purge(true); err != nil {
		ad.log.Printf("[ERROR] Cannot purge Cache: %s\n",
			err.Error())
		return err
	}

	var tagMap map[model.Tag][]*model.Item

	if tagMap, err = ad.db.TagLinkGetMap(); err != nil {
		ad.log.Printf("[CRITICAL] Failed to load Tag map: %s\n",
			err.Error())
		return err
	}

	// nolint: nilaway
	for tag, items := range tagMap {
		for _, item := range items {
			var (
				lng = ad.lngMap[item.FeedID]
				s   = ad.advisors[lng]
			)

			if err = s.Learn(tag.IDStr(), item.Strip()); err != nil { // nolint: nilaway
				ad.log.Printf("[ERROR] Failed to train Advisor on Item %q (%d): %s\n",
					item.Title,
					item.ID,
					err.Error())
			}
		}
	}

	ad.log.Println("[TRACE] Advisor has been retrained successfully.")

	return nil
} // func (ad *Advisor) Reset() error

// Learn teaches the Advisor about the link between a Tag and an Item.
func (ad *Advisor) Learn(tag *model.Tag, item *model.Item) error {
	var (
		err     error
		lng     string
		s       shield.Shield
		waitCnt int
	)

	ad.lock.Lock()
	defer ad.lock.Unlock()

	lng = ad.lngMap[item.FeedID]
	s = ad.advisors[lng]

CLASSIFY:
	// nolint: nilaway
	if err = s.Learn(tag.IDStr(), item.Strip()); err != nil {
		if worthARetry(err) && waitCnt < maxWait {
			waitCnt++
			waitForRetry()
			goto CLASSIFY
		}
		ad.log.Printf("[ERROR] Failed to train Tag Advisor on Item %q (%d): %s\n",
			item.Title,
			item.ID,
			err.Error())
		return err
	} else if err = ad.acache.Delete(item.IDStr()); err != nil {
		ad.log.Printf("[ERROR] Failed to delete cached Tag scores for Item %q (%d): %s\n",
			item.Title,
			item.ID,
			err.Error())
		return err
	}

	return nil
} // func (ad *Advisor) Learn(tag *model.Tag, item *model.Item) error

// Unlearn tells the Advisor to forget about the link between a Tag and an Item.
func (ad *Advisor) Unlearn(tag *model.Tag, item *model.Item) error {
	var (
		err     error
		lng     string
		s       shield.Shield
		waitCnt int
	)

	ad.lock.Lock()
	defer ad.lock.Unlock()

	lng = ad.lngMap[item.FeedID]
	s = ad.advisors[lng]

CLASSIFY:
	// nolint: nilaway
	if err = s.Forget(tag.IDStr(), item.Strip()); err != nil {
		if worthARetry(err) && waitCnt < maxWait {
			waitCnt++
			waitForRetry()
			goto CLASSIFY
		}
		ad.log.Printf("[ERROR] Failed to forget about Item %q (%d) and Tag %s (%d): %s\n",
			item.Title,
			item.ID,
			tag.Name,
			tag.ID,
			err.Error())
		return err
	} else if ad.acache.Delete(item.IDStr()); err != nil {
		ad.log.Printf("[ERROR] Failed to remove Tag scores for Item %q (%d) from Cache: %s\n",
			item.Title,
			item.ID,
			err.Error())
	}

	return nil
} // func (ad *Advisor) Unlearn(tag *model.Tag, item *model.Item) error

// Score calculates how strongly the Item matches different tags.
func (ad *Advisor) Score(item *model.Item) (SuggList, error) {
	var (
		err     error
		matches map[string]float64
		lng     string
		s       shield.Shield
		waitCnt int
		tags    []*model.Tag
		tmap    map[int64]bool
	)

	ad.lock.RLock()
	defer ad.lock.RUnlock()

	if tags, err = ad.db.TagLinkGetByItem(item); err != nil {
		ad.log.Printf("[ERROR] Failed to load Tags attached to Item %q (%d): %s\n",
			item.Title,
			item.ID,
			err.Error())
		return nil, err
	}

	tmap = make(map[int64]bool, len(tags))
	for _, tag := range tags {
		tmap[tag.ID] = true
	}

	lng = ad.lngMap[item.FeedID]
	s = ad.advisors[lng]

CLASSIFY:
	// nolint: nilaway
	if matches, err = s.Score(item.Strip()); err != nil {
		if worthARetry(err) && waitCnt < maxWait {
			waitCnt++
			waitForRetry()
			goto CLASSIFY
		}
		ad.log.Printf("[ERROR] Failed to calculate Tag scores for Item %q (%d): %s\n",
			item.Title,
			item.ID,
			err.Error())
		return nil, err
	}

	var lst = make(SuggList, 0, len(matches))

	for tname, degree := range matches {
		var sugg Suggestion

		sugg.TagID, _ = strconv.ParseInt(tname, 10, 64)

		if !tmap[sugg.TagID] {
			sugg.Match = degree
			lst = append(lst, sugg)
		}

	}

	slices.SortFunc(lst, cmpSugg)

	if len(lst) > 10 {
		return lst[:10], nil
	}

	return lst, nil
} // func (ad *Advisor) Score(item *model.Item) (SuggList, error)

func cmpSugg(a, b Suggestion) int {
	return -cmp.Compare[float64](a.Match, b.Match)
}
