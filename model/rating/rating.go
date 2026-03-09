// /home/krylon/go/src/github.com/blicero/newsroom/model/rating/rating.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-09 13:50:33 krylon>

// Package rating defines constants for rating news Items.
package rating

//go:generate stringer -type=Rating

// Rating identifies a rating of a news Item.
type Rating uint8

const (
	Unrated Rating = iota
	Boring
	Interesting
)
