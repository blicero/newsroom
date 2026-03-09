// /home/krylon/go/src/github.com/blicero/newsroom/model/model.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-09 13:51:16 krylon>

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
