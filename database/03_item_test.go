// /home/krylon/go/src/github.com/blicero/newsroom/database/03_item_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-08 12:33:47 krylon>

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
