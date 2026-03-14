// /home/krylon/go/src/github.com/blicero/newsroom/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-14 14:12:40 krylon>

// Package model defines data types that are used throughout the application.
package model

import (
	"net/url"
	"time"

	"github.com/blicero/newsroom/model/rating"
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
	Paused          bool
}

// IsDue returns true if the feed is due for a refresh.
func (f *Feed) IsDue() bool {
	return f.LastRefresh.Add(f.RefreshInterval).Before(time.Now())
} // func (f *Feed) IsDue() bool

// Item is a news article, blog post, etc.
type Item struct {
	ID        int64
	FeedID    int64
	Title     string
	URL       *url.URL
	Rating    rating.Rating
	Timestamp time.Time
	Body      string
}

// IsRated returns true if the Item has been rated.
func (i *Item) IsRated() bool {
	return i.Rating != rating.Unrated
} // func (i *Item) IsRated() bool

// IsBoring returns true if the Item has been rated as Boring.
func (i *Item) IsBoring() bool {
	return i.Rating == rating.Boring
} // func (i *Item) IsBoring() bool

// Tag is a descriptive bit of text we can attach to Items.
type Tag struct {
	ID   int64
	Name string
}

// TagLink attaches a Tag to an Item.
type TagLink struct {
	ID     int64
	TagID  int64
	ItemID int64
}
