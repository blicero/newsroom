// /home/krylon/go/src/github.com/blicero/newsroom/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-09 13:48:08 krylon>

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
	FeedSetPause
	FeedDelete
	ItemAdd
	ItemGetByID
	ItemGetByFeed
	ItemGetAll
)
