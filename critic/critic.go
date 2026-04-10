// /home/krylon/go/src/github.com/blicero/newsroom/critic/critic.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-10 12:59:30 krylon>

// Package critic deals with guessing the most probable rating for Items.
// Like a spam filter for news.
package critic

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/blicero/newsroom/cache"
	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
	"github.com/blicero/shield"
)

const (
	cacheTimeout = time.Minute * 240 // TODO I'll increase this value once I'm done testing.
	backoffDelay = time.Millisecond * 250
	errTmp       = "resource temporarily unavailable"
)

var languages = []string{"en", "de"}

type score struct {
	ID     int64
	Rating rating.Rating
}

// Critic passes judgement on news Items, whether they be
// Interesting or Boring.
type Critic struct {
	log     *log.Logger
	critics map[string]shield.Shield
	db      *database.Database
	lock    sync.RWMutex
	rcache  *cache.Cache[score]
	lngMap  map[int64]string
}

// New creates and returns a fresh Critic instance.
func New() (*Critic, error) {
	var (
		err error
		c   = &Critic{
			critics: make(map[string]shield.Shield),
			lngMap:  make(map[int64]string),
		}
	)

	if c.log, err = common.GetLogger(logdomain.Critic); err != nil {
		return nil, err
	} else if c.db, err = database.Open(common.DbPath); err != nil {
		c.log.Printf("[CRITICAL] Failed to open database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	} else if c.rcache, err = cache.New[score]("critic_rating"); err != nil {
		c.log.Printf("[ERROR] Failed to open rating cache: %s\n",
			err.Error())
		return nil, err
	}

	for _, lng := range languages {
		var (
			tok       shield.Tokenizer
			storePath = filepath.Join(common.CachePath, fmt.Sprintf("critic_store_%s", lng))
			store     = shield.NewLevelDBStore(storePath)
		)

		switch lng {
		case "en":
			tok = shield.NewEnglishTokenizer()
		case "de":
			tok = shield.NewGermanTokenizer()
		default:
			c.log.Printf("[CANTHAPPEN] Unsupported language %s\n",
				lng)
			return nil, fmt.Errorf("unsupported language %s", lng)
		}

		c.critics[lng] = shield.New(tok, store)
	}

	return c, nil
} // func New() (*Critic, error)

// Reset discards the Critic's model training state and trains it again
// based on the rated Items in the Database.
func (c *Critic) Reset() error {
	var (
		err   error
		items []*model.Item
		feeds []*model.Feed
	)

	c.lock.Lock()
	defer c.lock.Unlock()

	if feeds, err = c.db.FeedGetAll(); err != nil {
		c.log.Printf("[ERROR] Failed to load Feeds from Database: %s\n",
			err.Error())
		return err
	}

	c.lngMap = make(map[int64]string, len(feeds))

	for _, f := range feeds {
		c.lngMap[f.ID] = f.Language
	}

	if items, err = c.db.ItemGetRated(); err != nil {
		c.log.Printf("[ERROR] Failed to load rated Items from Database: %s\n",
			err.Error())
		return err
	}

	for lng, s := range c.critics {
		c.log.Printf("[TRACE] Resetting Shield for %s\n",
			lng)
		if err = s.Reset(); err != nil {
			c.log.Printf("[ERROR] Failed to reset Shield for %s: %s\n",
				lng,
				err.Error())
			return err
		}
	}

	// nolint: nilaway
	for _, item := range items {
		if item == nil {
			continue
		}

		var (
			lng = c.lngMap[item.FeedID]
			s   = c.critics[lng]
		)

		if err = s.Learn(item.Rating.String(), item.Strip()); err != nil {
			c.log.Printf("[ERROR] Failed to learn about Item %d (%s): %s\n",
				item.ID,
				item.Title,
				err.Error())
		}
	}

	return nil
} // func (c *Critic) Reset() error

// Learn teaches the model about an Item.
func (c *Critic) Learn(item *model.Item) error {
	var (
		err error
		lng string
		s   shield.Shield
	)

	if lng, err = c.getLanguage(item); err != nil {
		c.log.Printf("[ERROR] Failed to look up language for Item %s: %s\n",
			item.Title,
			err.Error())
		return err
	} else if lng == "" {
		c.log.Printf("[ERROR] Failed to look up language for Item %s, falling back to English by default.\n",
			item.Title)
		lng = "en"
	}

	if s = c.critics[lng]; s == nil {
		err = fmt.Errorf("no Critic found for language %s",
			lng)
		c.log.Printf("[ERROR] %s\n", err.Error())
		return err
	} else if err = s.Learn(item.Rating.String(), item.Strip()); err != nil {
		c.log.Printf("[ERROR] Failed to learn about Item %d (%s): %s\n",
			item.ID,
			item.Title,
			err.Error())
		return err
	}

	return nil
} // func (c *Critic) Learn(item *model.Item) error

// Unlearn removes an Item from the learniung corpus.
func (c *Critic) Unlearn(item *model.Item) error {
	var (
		err error
		lng string
		s   shield.Shield
	)

	if lng, err = c.getLanguage(item); err != nil {
		c.log.Printf("[ERROR] Failed to look up language for Item %s: %s\n",
			item.Title,
			err.Error())
		return err
	} else if lng == "" {
		c.log.Printf("[ERROR] Failed to look up language for Item %s, falling back to English by default.\n",
			item.Title)
		lng = "en"
	}

	if s = c.critics[lng]; s == nil {
		err = fmt.Errorf("no Critic found for language %s",
			lng)
		c.log.Printf("[ERROR] %s\n", err.Error())
		return err
	} else if err = s.Forget(item.Rating.String(), item.Strip()); err != nil {
		c.log.Printf("[ERROR] Failed to learn about Item %d (%s): %s\n",
			item.ID,
			item.Title,
			err.Error())
		return err
	}

	return nil
} // func (c *Critic) Unlearn(item *model.Item) error

// Classify attempts to guess a Rating for the given Item.
func (c *Critic) Classify(item *model.Item) (rating.Rating, error) {
	var (
		err      error
		lng, cls string
		s        shield.Shield
		r        rating.Rating
	)

	if lng, err = c.getLanguage(item); err != nil {
		c.log.Printf("[ERROR] Failed to look up language for Item %s: %s\n",
			item.Title,
			err.Error())
		return r, err
	} else if lng == "" {
		c.log.Printf("[ERROR] Failed to look up language for Item %s, falling back to English by default.\n",
			item.Title)
		lng = "en"
	}

	if s = c.critics[lng]; s == nil {
		err = fmt.Errorf("no Critic found for language %s",
			lng)
		c.log.Printf("[ERROR] %s\n", err.Error())
		return r, err
	}

	if cls, err = s.Classify(item.Strip()); err != nil {
		c.log.Printf("[ERROR] Failed to classify Item %q (%d): %s\n",
			item.Title,
			item.ID,
			err.Error())
		return r, err
	} else if r, err = rating.FromString(cls); err != nil {
		c.log.Printf("[ERROR] Failed to convert class %q to Rating: %s\n",
			cls,
			err.Error())
		return rating.Unrated, err
	}

	item.GuessedRating = r

	return r, nil
} // func (c *Critic) Classify(item *model.Item) error

func (c *Critic) getLanguage(item *model.Item) (string, error) {
	var (
		err  error
		feed *model.Feed
	)

	c.lock.RLock()

	if lng, ok := c.lngMap[item.FeedID]; ok {
		c.lock.RUnlock()
		return lng, nil
	}

	c.lock.RUnlock()
	c.lock.Lock()
	defer c.lock.Unlock()

	if feed, err = c.db.FeedGetByID(item.FeedID); err != nil {
		c.log.Printf("[ERROR] Failed to look up Feed %d: %s\n",
			item.FeedID,
			err.Error())
		return "", err
	} else if feed == nil {
		err = fmt.Errorf("feed %d was not found in database",
			item.FeedID)
		c.log.Printf("[CANTHAPPEN] %s\n",
			err.Error())
		return "", err
	}

	c.lngMap[feed.ID] = feed.Language
	return feed.Language, nil
} // func (c *Critic) getLanguage(item *model.Item) (string, error)
