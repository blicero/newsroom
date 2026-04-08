// /home/krylon/go/src/github.com/blicero/newsroom/database/02_database_feed_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-08 12:44:06 krylon>

package database

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/model"
)

const testFeedCnt = 16

var feeds [testFeedCnt]*model.Feed

func TestFeedAd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	const homepage = "https://www.example.com/"

	for i := range testFeedCnt {
		var (
			err  error
			ustr string
			feed = &model.Feed{
				Name:            fmt.Sprintf("Feed%02d", i),
				Language:        "en",
				RefreshInterval: time.Second * 3600,
			}
		)

		ustr = fmt.Sprintf("https://www.example.com/feeds/feed%02d.xml", i)

		if feed.URL, err = url.Parse(ustr); err != nil {
			t.Errorf("Failed to parse URL of Feed %02d (%q): %s",
				i,
				ustr,
				err.Error())
			continue
		} else if feed.Homepage, err = url.Parse(homepage); err != nil {
			t.Errorf("Failed to parse Homepage %q: %s",
				homepage,
				err.Error())
			continue
		} else if err = tdb.FeedAdd(feed); err != nil {
			t.Errorf("Failed to add Feed %02d to database: %s",
				i,
				err.Error())
			continue

		} else if feed.ID == 0 {
			t.Errorf("FeedAdd did not return an error, but ID of %s is 0",
				feed.Name)
			continue
		}

		feeds[i] = feed
	}
} // func TestFeedAd(t *testing.T)

func TestFeedGetAll(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err     error
		dbFeeds []*model.Feed
	)

	if dbFeeds, err = tdb.FeedGetAll(); err != nil {
		t.Fatalf("FeedGetAll failed: %s", err.Error())
	} else if len(dbFeeds) != testFeedCnt {
		t.Fatalf("FeedGetAll returned unexpected number of Feeds: %d (expected %d)",
			len(dbFeeds),
			testFeedCnt)
	}
} // func TestFeedGetAll(t *testing.T)

func TestFeedGetDue(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err     error
		dbFeeds []*model.Feed
	)

	if dbFeeds, err = tdb.FeedGetDue(); err != nil {
		t.Fatalf("FeedGetAll failed: %s", err.Error())
	} else if len(dbFeeds) != testFeedCnt {
		t.Fatalf("FeedGetAll returned unexpected number of Feeds: %d (expected %d)",
			len(dbFeeds),
			testFeedCnt)
	}
} // func TestFeedGetDue(t *testing.T)

func TestFeedSetLastRefresh(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err error
		now = time.Now()
	)

	for _, feed := range feeds {
		if err = tdb.FeedSetLastRefresh(feed, now); err != nil {
			t.Errorf("Failed to update refresh timestamp of Feed %s: %s",
				feed.Name,
				err.Error())
		} else if !feed.LastRefresh.Equal(now) {
			t.Errorf("After updating refresh timestamp in database, timestamp in Feed instance is not changed:\nExpected: %s\nGot: %s\n",
				now.Format(common.TimestampFormat),
				feed.LastRefresh.Format(common.TimestampFormat))
		}
	}

} // func TestFeedSetLastRefresh(t *testing.T)

func TestFeedGetDueAgain(t *testing.T) {
	// After updating all those refresh timestamp, no FeedGetDue should
	// return no Feeds.
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err     error
		dbFeeds []*model.Feed
	)

	if dbFeeds, err = tdb.FeedGetDue(); err != nil {
		t.Fatalf("FeedGetAll failed: %s", err.Error())
	} else if len(dbFeeds) != 0 {
		t.Fatalf("FeedGetAll returned unexpected number of Feeds: %d (expected 0)",
			len(dbFeeds))
	}
} // func TestFeedGetDueAgain(t *testing.T)
