// /home/krylon/go/src/github.com/blicero/newsroom/critic/critic.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-22 17:17:03 krylon>

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
}

// New creates and returns a fresh Critic instance.
func New() (*Critic, error) {
	var (
		err error
		c   = new(Critic)
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

// Retrain discards the Critic's model training state and trains it again
// based on the rated Items in the Database.
func (c *Critic) Retrain() error {
	var (
		err    error
		items  []*model.Item
		lngMap map[int64]string
		feeds  []*model.Feed
	)

	if feeds, err = c.db.FeedGetAll(); err != nil {
		c.log.Printf("[ERROR] Failed to load Feeds from Database: %s\n",
			err.Error())
		return err
	}

	lngMap = make(map[int64]string, len(feeds))

	for _, f := range feeds {
		lngMap[f.ID] = f.Language
	}

	if items, err = c.db.ItemGetRated(); err != nil {
		c.log.Printf("[ERROR] Failed to load rated Items from Database: %s\n",
			err.Error())
		return err
	}

	// for lng, s := range c.critics {
	// 	c.log.Printf("[TRACE] Resetting Shield for %s\n",
	// 		lng)
	// 	if err = s.Reset(); err != nil {
	// 		c.log.Printf("[ERROR] Failed to reset Shield for %s: %s\n",
	// 			lng,
	// 			err.Error())
	// 		return err
	// 	}
	// }

	// for _, item := range items {
	// 	var (
	// 		lng = lngMap[item.FeedID]
	// 		s   = c.critics[lng]
	// 	)

	// 	//if err = s.Learn(item.Rating.String(),
	// }

	return nil
} // func (c *Critic) Retrain() error
