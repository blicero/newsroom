// /home/krylon/go/src/github.com/blicero/newsroom/engine/engine.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-12 13:29:59 krylon>

// Package engine defines the Engine that manages the subscriptions.
package engine

import (
	"fmt"
	"log"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"github.com/mmcdole/gofeed"
)

const checkRefreshInterval = time.Minute

// Engine implements the handling of subcriptions, regularly updating Feeds and
// storing new Items in the Database.
type Engine struct {
	log        *log.Logger
	pool       *database.Pool
	active     atomic.Bool
	refreshing atomic.Bool
	workerCnt  int
}

// Create creates and returns a new Engine.
func Create(cnt int) (*Engine, error) {
	var (
		err   error
		psize int
		eng   = &Engine{workerCnt: cnt}
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

	return eng, nil
} // func Create(cnt int64) (*Engine, error)

// IsActive returns the state of the Engine's active flag.
func (eng *Engine) IsActive() bool {
	return eng.active.Load()
} // func (eng *Engine) IsActive() bool

// Stop clears the Engine's active flag.
func (eng *Engine) Stop() {
	eng.active.Store(false)
} // func (eng *Engine) Stop()

// Start sets the Engine's parts into motion.
func (eng *Engine) Start() {
	if eng.IsActive() {
		return
	}

	go eng.supervisor()
} // func (eng *Engine) Start()

func (eng *Engine) supervisor() {
	eng.active.Store(true)
	defer eng.active.Store(false)

	eng.log.Printf("[INFO] Engine supervisor starting up.\n")

	var activeTicker = time.NewTicker(common.ActiveTimeout)
	defer activeTicker.Stop()

	var refreshTicker = time.NewTicker(checkRefreshInterval)
	defer refreshTicker.Stop()

	for eng.IsActive() {
		select {
		case <-activeTicker.C:
			continue
		case <-refreshTicker.C:
			// Trigger a refresh
			go eng.performRefresh()
		}
	}
} // func (eng *Engine) supervisor()

func (eng *Engine) performRefresh() {
	var (
		err   error
		db    *database.Database
		feeds []*model.Feed
	)

	if !eng.refreshing.CompareAndSwap(false, true) {
		eng.log.Printf("[TRACE] Refresh is already going on, I'm quitting.\n")
		return
	}

	defer eng.refreshing.Store(false)

	eng.log.Printf("[INFO] Check Feeds for refresh.\n")

	db = eng.pool.Get()
	defer eng.pool.Put(db)

	if feeds, err = db.FeedGetDue(); err != nil {
		eng.log.Printf("[ERROR] Failed to get Feeds for refresh: %s\n",
			err.Error())
		return
	} else if len(feeds) == 0 {
		eng.log.Printf("[DEBUG] No feeds need a refresh. Bye.\n")
		return
	}

	var (
		qsize = max(len(feeds), eng.workerCnt)
		feedQ = make(chan *model.Feed, qsize)
		itemQ = make(chan *model.Item, qsize*2)
		wg    sync.WaitGroup
	)

	wg.Add(eng.workerCnt)

	go eng.itemWorker(itemQ)
	for i := range eng.workerCnt {
		go eng.refreshWorker(i, feedQ, itemQ, &wg)
	}

	for _, feed := range feeds {
		feedQ <- feed
	}

	close(feedQ)

	wg.Wait()

	close(itemQ)
} // func (eng *Engine) performRefresh()

func (eng *Engine) refreshWorker(id int, feedQ <-chan *model.Feed, itemQ chan<- *model.Item, wg *sync.WaitGroup) {
	defer wg.Done()
	eng.log.Printf("[TRACE] refreshWorker %02d starting up.\n", id)
	defer eng.log.Printf("[TRACE] refreshWorker %02d is finished.\n", id)

	var (
		err    error
		parser = gofeed.NewParser()
	)

	parser.UserAgent = fmt.Sprintf("%s %s",
		common.AppName,
		common.Version)

	for feed := range feedQ {
		// Fetch feed, process Items, send items through queue.
		eng.log.Printf("[DEBUG] refreshWorker %02d refresh %q\n",
			id,
			feed.Name)
		var gfeed *gofeed.Feed

		if gfeed, err = parser.ParseURL(feed.URL.String()); err != nil {
			eng.log.Printf("[ERROR] refreshWorker %02d failed to fetch %s (%s): %s\n",
				id,
				feed.Name,
				feed.URL,
				err.Error())
			continue
		}

		eng.log.Printf("[TRACE] refreshWorker %02d processing %d Items from %s\n",
			id,
			len(gfeed.Items),
			feed.Name)

		for _, gitem := range gfeed.Items {
			var item = &model.Item{
				FeedID:    feed.ID,
				Title:     gitem.Title,
				Timestamp: *gitem.UpdatedParsed,
				Body:      gitem.Description,
			}

			if item.URL, err = url.Parse(gitem.Link); err != nil {
				eng.log.Printf("[ERROR] refreshWorker %02d failed to parse URL from item %q (%q): %s\n",
					id,
					gitem.Title,
					gitem.Link,
					err.Error())
				continue
			}

			itemQ <- item
		}
	}
} // func (eng *Engine) refreshWorker(id int)

func (eng *Engine) itemWorker(itemQ <-chan *model.Item) {
	eng.log.Printf("[TRACE] itemWorker starting up.\n")
	defer eng.log.Printf("[TRACE] itemWorker is finished.\n")

	var (
		err error
		db  *database.Database
	)

	db = eng.pool.Get()
	defer eng.pool.Put(db)

	for item := range itemQ {
		var dbItem *model.Item

		if dbItem, err = db.ItemGetByURL(item.URL); err != nil {
			eng.log.Printf("[ERROR] Failed to check if Item %q already exists in Database: %s\n",
				item.Title,
				err.Error())
			continue
		} else if dbItem != nil {
			continue
		} else if err = db.ItemAdd(item); err != nil {
			eng.log.Printf("[ERROR] Failed to add Item %q (%s) to Database: %s\n",
				item.Title,
				item.URL,
				err.Error())
			continue
		}
	}
} // func (eng *Engine) itemWorker(itemQ <-chan *model.Item)
