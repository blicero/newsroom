// /home/krylon/go/src/github.com/blicero/newsroom/logdomain/logdomain.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-01 15:25:55 krylon>

package logdomain

//go:generate stringer -type=ID

// ID signifies a part of the application that wants to write to the log.
type ID uint8

const (
	Database ID = iota
	DBPool
	Engine
	Cache
	Critic
	Classifier
	Scrub
	Web
	Main
)

// All returns a slice of all valid ID values.
func All() []ID {
	return []ID{
		Database,
		DBPool,
		Engine,
		Cache,
		Critic,
		Classifier,
		Scrub,
		Web,
		Main,
	}
} // func All() []ID
