// /home/krylon/go/src/github.com/blicero/newsroom/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-08 14:37:05 krylon>

// Package model defines data types that are used throughout the application.
package model

import (
	"net/url"
	"strconv"
	"time"

	"github.com/blicero/newsroom/model/rating"
	"github.com/darkoatanasovski/htmltags"
)

// Feed is an RSS feed we can subscribe to.
type Feed struct {
	ID              int64
	Name            string
	Language        string
	URL             *url.URL
	Homepage        *url.URL
	RefreshInterval time.Duration
	LastRefresh     time.Time
	Active          bool
}

// IsDue returns true if the feed is due for a refresh.
func (f *Feed) IsDue() bool {
	return f.LastRefresh.Add(f.RefreshInterval).Before(time.Now())
} // func (f *Feed) IsDue() bool

// Item is a news article, blog post, etc.
type Item struct {
	ID            int64
	FeedID        int64
	Title         string
	URL           *url.URL
	Rating        rating.Rating
	GuessedRating rating.Rating
	Timestamp     time.Time
	Body          string
	stripped      string
}

// IsRated returns true if the Item has been rated.
func (i *Item) IsRated() bool {
	return i.Rating != rating.Unrated
} // func (i *Item) IsRated() bool

// IsBoring returns true if the Item has been rated as Boring.
func (i *Item) IsBoring() bool {
	return i.Rating == rating.Boring
} // func (i *Item) IsBoring() bool

// EffectiveRating returns an Item's guessed Rating if it is unrated,
// otherwise the Item's Rating.
func (i *Item) EffectiveRating() rating.Rating {
	if i.Rating == rating.Unrated {
		return i.GuessedRating
	}

	return i.Rating
} // func (i *Item) EffectiveRating() rating.Rating

// Strip returns the Item's Title + Body, stripped of all HTML elements.
// The result is cached, subsequent calls return the cached value.
//
// CAVEAT: Caching is result per Item only at this point.
func (i *Item) Strip() string {
	if i.stripped != "" {
		return i.stripped
	}

	var (
		err   error
		long  string
		nodes htmltags.Nodes
	)

	long = i.Title + " " + i.Body

	if nodes, err = htmltags.Strip(long, nil, true); err != nil {
		panic(err)
	}

	i.stripped = nodes.ToString()
	return i.stripped
} // func (i *Item) Strip() string

// Tag is a descriptive bit of text we can attach to Items.
type Tag struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parent"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	FullName string `json:"full_name"`
}

// Parent returns the Tags ParentID as a string, or an empty string
// if ParentID is 0.
func (t *Tag) Parent() string {
	if t.ParentID == 0 {
		return ""
	}

	return strconv.FormatInt(t.ParentID, 10)
} // func (t *Tag) Parent() string

// TagLink attaches a Tag to an Item.
type TagLink struct {
	ID     int64
	TagID  int64
	ItemID int64
}
