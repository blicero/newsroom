// /home/krylon/go/src/github.com/blicero/newsroom/database/03_item_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-22 12:31:57 krylon>

package database

import (
	"fmt"
	"math/rand"
	"net/url"
	"testing"
	"time"

	"github.com/blicero/newsroom/model"
)

const itemCnt = 32

var titems [itemCnt]*model.Item

func TestItemAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var err error

	for idx := range itemCnt {
		var (
			ustr = fmt.Sprintf("https://www.example.com/news/item%02d", idx+1)
			item = &model.Item{
				FeedID:    feeds[rand.Intn(testFeedCnt)].ID,
				Title:     fmt.Sprintf("TestItem %02d", idx+1),
				Timestamp: time.Now(),
				Body:      fmt.Sprintf("Bla Bla Bla %02d", idx+1),
			}
		)

		if item.URL, err = url.Parse(ustr); err != nil {
			t.Fatalf("Failed to parse URL %q: %s",
				ustr,
				err.Error())
		} else if err = tdb.ItemAdd(item); err != nil {
			t.Fatalf("Failed to add Item %s to Database: %s",
				item.Title,
				err.Error())
		} else if item.ID == 0 {
			t.Fatalf("Item %s was added with out an error, but its ID is still 0",
				item.Title)
		}

		titems[idx] = item
	}
} // func TestItemAdd(t *testing.T)

func TestItemStrip(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	} else if len(titems) != itemCnt {
		t.SkipNow()
	}

	for _, item := range titems {
		var (
			err      error
			stripped string
		)

		if stripped, err = tdb.Strip(item); err != nil {
			t.Errorf("Failed to strip Item %q (%d): %s",
				item.Title,
				item.ID,
				err.Error())
		} else if stripped == "" {
			t.Errorf("Stripping Item %q (%d) returned an empty string",
				item.Title,
				item.ID)
		}
	}
} // func TestItemStrip(t *testing.T)
