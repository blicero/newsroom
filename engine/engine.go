// /home/krylon/go/src/github.com/blicero/newsroom/engine/engine.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-10 15:17:42 krylon>

// Package engine defines the Engine that manages the subscriptions.
package engine

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
)

// Engine implements the handling of subcriptions, regularly updating Feeds and
// storing new Items in the Database.
type Engine struct {
	log       *log.Logger
	pool      *database.Pool
	active    atomic.Bool
	workerCnt atomic.Int64
	wg        sync.WaitGroup
	fetchQ    chan *model.Feed
}

func Create(cnt int) (*Engine, error) {
	var (
		err   error
		psize int
		eng   = new(Engine)
	)

	psize = max(cnt/2, 2)

	if eng.log, err = common.GetLogger(logdomain.Engine); err != nil {
		return nil, err
	} else if eng.pool, err = database.NewPool(psize); err != nil {
		eng.log.Printf("[CRITICAL] Cannot open DB connection pool(%d): %s\n",
			cnt,
			err.Error())
		return nil, err
	}

	eng.fetchQ = make(chan *model.Feed, cnt)

	return eng, nil
} // func Create(cnt int64) (*Engine, error)

// IsActive returns the state of the Engine's active flag.
func (eng *Engine) IsActive() bool {
	return eng.active.Load()
} // func (eng *Engine) IsActive() bool

// Stop clears the Engine's active flag.
func (eng *Engine) Stop() {
	eng.active.Store(false)
	eng.wg.Wait()
} // func (eng *Engine) Stop()

func (eng *Engine) worker(id int64) {
	eng.wg.Add(1)
	defer eng.wg.Done()

	var ticker = time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for eng.active.Load() {
		<-ticker.C
	}
} // func (eng *Engine) worker(id int64)

func (eng *Engine) refreshFeeds() {
	var (
		err   error
		db    *database.Database
		feeds []*model.Feed
	)

	db = eng.pool.Get()
	defer eng.pool.Put(db)

	if feeds, err = db.FeedGetDue(); err != nil {
		eng.log.Printf("[ERROR] Failed to load Feeds due for a refresh: %s\n",
			err.Error())
		return
	}

} // func (eng *Engine) refreshFeeds()
