// /home/krylon/go/src/github.com/blicero/newsroom/database/07_bookmark_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-04 12:52:32 krylon>

package database

import (
	"testing"
	"time"

	"github.com/blicero/newsroom/model"
)

var bookmarks = make([]*model.Bookmark, 0)

func TestBookmarkAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	for _, item := range titems {
		var (
			err      error
			bookmark = &model.Bookmark{
				ItemID:   item.ID,
				Deadline: time.Now().Add(time.Hour * 24),
			}
		)

		if err = tdb.BookmarkAdd(bookmark); err != nil {
			t.Fatalf("Cannot add bookmark for Item %q (%d): %s",
				item.Title,
				item.ID,
				err.Error())
		}

		bookmarks = append(bookmarks, bookmark)
	}
} // func TestBookmarkAdd(t *testing.T)

func TestBookmarkGetAll(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err    error
		tmarks []*model.Bookmark
	)

	if tmarks, err = tdb.BookmarkGetAll(); err != nil {
		t.Fatalf("Failed to load all bookmarks: %s",
			err.Error())
	} else if len(tmarks) != len(bookmarks) {
		t.Fatalf("Unexpected number of bookmarks: Expected %d, got %d",
			len(bookmarks),
			len(tmarks))
	}

} // func TestBookmarkGetAll(t *testing.T)
