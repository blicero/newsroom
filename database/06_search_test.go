// /home/krylon/go/src/github.com/blicero/newsroom/database/06_search_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-20 14:00:14 krylon>

package database

import (
	"testing"
	"time"

	"github.com/blicero/newsroom/model"
)

func TestSearch(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	type testCase struct {
		name  string
		parm  SearchParms
		err   bool
		empty bool
	}

	var cases = []testCase{
		{
			name: "plain",
			parm: SearchParms{
				Query: "%Bla%",
			},
		},
		{
			name: "by date",
			parm: SearchParms{
				Query: "%Bla%",
				DateP: true,
				DateRange: [2]time.Time{
					time.Now().Add(time.Second * -86400),
					time.Now().Add(time.Second * 86400),
				},
			},
		},
		{
			name: "by date reverse",
			parm: SearchParms{
				Query: "%Bla%",
				DateP: true,
				DateRange: [2]time.Time{
					time.Now().Add(time.Second * 86400),
					time.Now().Add(time.Second * -86400),
				},
			},
			empty: true,
		},
		{
			name: "by tags",
			parm: SearchParms{
				Query: "%Bla%",
				TagP:  true,
				Tags: map[int64]bool{
					tags[0].ID: true,
					tags[1].ID: true,
					tags[2].ID: true,
					tags[3].ID: true,
				},
			},
		},
	}

	for _, c := range cases {
		var (
			err   error
			items []*model.Item
		)

		if items, err = tdb.Search(&c.parm); err != nil {
			if c.err {
				continue
			}

			t.Errorf("Search %s failed: %s",
				c.name,
				err.Error())
		} else if c.err {
			t.Errorf("Search %s should have resulted in error, but didn't",
				c.name)
		} else if len(items) == 0 && !c.empty {
			t.Errorf("Search %s returned no items",
				c.name)
		}
	}
} // func TestSearch(t *testing.T)
