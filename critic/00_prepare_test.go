// /home/krylon/go/src/github.com/blicero/newsroom/critic/00_prepare_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 30. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-30 20:21:58 krylon>

package critic

import (
	"fmt"
	"net/url"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
)

func purl(s string) *url.URL {
	if u, err := url.Parse(s); err != nil {
		panic(err)
	} else {
		return u
	}
} // func purl(s string) *url.URL

func addItems(feed *model.Feed, db *database.Database) error {
	type titem struct {
		title string
		body  string
		cls   rating.Rating
	}

	var items = []titem{
		{
			title: "Linux again dominant system on supercomputers",
			body: `
The free operating system once again dominates the supercomputer market,
with over ninety-five percent of the computers on the TOP500 list running
some variant of it.
`,
			cls: rating.Interesting,
		},
		{
			title: "Olympic games end with a spectacle",
			body: `
The Olympic games in Italy ended tonight with a spectacular show
and fireworks. Many athletes participated in front of a worldwide
audience numbering in the hundreds of millions.
`,
			cls: rating.Boring,
		},
		{
			title: "Nobel prize for physics awarded for research on dark matter",
			body: `
The Nobel prize for physics goes to two scientists from Japan this year, for their
contributions to understanding the nature and distribution of dark matter in the
universe.
`,
			cls: rating.Interesting,
		},
	}

	var (
		err   error
		tbase = time.Now().Add(time.Second * -86400)
	)

	for idx, tmpl := range items {
		var item = model.Item{
			FeedID:    feed.ID,
			Title:     tmpl.title,
			URL:       purl(fmt.Sprintf("https://www.example.com/news/item%03d", idx)),
			Timestamp: tbase.Add(time.Second * time.Duration(idx) * 30),
			Body:      tmpl.body,
		}

		if err = db.ItemAdd(&item); err != nil {
			return err
		} else if err = db.ItemSetRating(&item, tmpl.cls); err != nil {
			return err
		}
	}

	return nil
} // func addItems(feed *model.Feed, db *database.Database) ([]*model.Item, error)

func prepare() error {
	var (
		err        error
		db         *database.Database
		feed       *model.Feed
		furl, hurl *url.URL
	)

	if furl, err = url.Parse("https://www.example.com/news/rss"); err != nil {
		return err
	} else if hurl, err = url.Parse("https://www.example.com/"); err != nil {
		return err
	} else if db, err = database.Open(common.DbPath); err != nil {
		return err
	}

	feed = &model.Feed{
		Name:            "Test 01",
		Language:        "en",
		URL:             furl,
		Homepage:        hurl,
		RefreshInterval: time.Second * 3600,
	}

	if err = db.FeedAdd(feed); err != nil {
		return err
	} else if err = addItems(feed, db); err != nil {
		return err
	}

	return nil
} // func prepare() error
