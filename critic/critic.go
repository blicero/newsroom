// /home/krylon/go/src/github.com/blicero/newsroom/critic/critic.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-19 17:22:46 krylon>

// Package critic deals with guessing the most probable rating for Items.
// Like a spam filter for news.
package critic

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/shield"
)

const (
	cacheTimeout = time.Minute * 240 // TODO I'll increase this value once I'm done testing.
	backoffDelay = time.Millisecond * 250
	errTmp       = "resource temporarily unavailable"
)

var languages = []string{"en", "de"}

// Critic passes judgement on news Items, whether they be
// Interesting or Boring.
type Critic struct {
	log     *log.Logger
	critics map[string]shield.Shield
	db      *database.Database
	lock    sync.RWMutex
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
