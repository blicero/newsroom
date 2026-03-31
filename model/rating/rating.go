// /home/krylon/go/src/github.com/blicero/newsroom/model/rating/rating.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-31 15:01:35 krylon>

// Package rating defines constants for rating news Items.
package rating

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=Rating

// Rating identifies a rating of a news Item.
type Rating uint8

const (
	Unrated Rating = iota
	Boring
	Interesting
)

// FromString returns the Rating expressed by the given string.
func FromString(s string) (Rating, error) {
	switch strings.ToLower(s) {
	case "unrated":
		return Unrated, nil
	case "unknown":
		return Unrated, nil
	case "boring":
		return Boring, nil
	case "interesting":
		return Interesting, nil
	default:
		return Unrated, fmt.Errorf("Invalid Rating: %q", s)
	}
} // func FromString(s string) (Rating, error)
