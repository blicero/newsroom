// /home/krylon/go/src/github.com/blicero/newsroom/config/config.go
// -*- mode: go; coding: utf-8; -*-
// Created on 29. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-06-27 13:33:51 krylon>

// Package config defines settings that can be modified by a user.
package config

// Config defines the variables that a user can set.
type Config struct {
	Debug bool
	Web   struct {
		Address    string
		HideBoring bool
	}
	Path struct {
		Base      string
		Log       string
		Database  string
		Cache     string
		Blacklist string
	}
}
