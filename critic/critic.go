// /home/krylon/go/src/github.com/blicero/newsroom/critic/critic.go
// -*- mode: go; coding: utf-8; -*-
// Created on 16. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-16 19:04:25 krylon>

// Package critic deals with guessing the most probable rating for Items.
// Like a spam filter for news.
package critic

import (
	"log"
	"sync"
	"time"

	"github.com/blicero/newsroom/database"
	"github.com/blicero/shield"
)

const (
	cacheTimeout = time.Minute * 240 // TODO I'll increase this value once I'm done testing.
	backoffDelay = time.Millisecond * 250
	errTmp       = "resource temporarily unavailable"
)

// Critic passes judgement on news Items, whether they be
// Interesting or Boring.
type Critic struct {
	log     *log.Logger
	critics map[string]shield.Shield
	db      *database.Database
	lock    sync.RWMutex
}
