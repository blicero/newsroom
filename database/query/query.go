// /home/krylon/go/src/github.com/blicero/newsroom/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-14 12:21:35 krylon>

// Package query defines symbolic constants to identify database queries.
package query

//go:generate stringer -type=ID

// ID identifies a database query
type ID uint8

const (
	FeedAdd ID = iota
	FeedGetByID
	FeedGetDue
	FeedGetAll
	FeedSetInterval
	FeedSetLastRefresh
	FeedSetPause
	FeedDelete
	ItemAdd
	ItemGetByID
	ItemGetByURL
	ItemGetByFeed
	ItemGetAll
	ItemCount
	TagAdd
	TagGetByID
	TagGetAll
	TagDelete
	TagLinkAdd
	TagLinkGetByTag
	TagLinkGetByItem
	TagLinkDelete
)
