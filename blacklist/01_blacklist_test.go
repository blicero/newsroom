// /home/krylon/go/src/github.com/blicero/newsroom/blacklist/01_blacklist_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 13. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-13 15:15:35 krylon>

package blacklist

import "testing"

var tbl *Blacklist

var testPatterns = []string{
	"heise\\+",
}

func TestCreateBlacklist(t *testing.T) {
	var err error

	if tbl, err = New(); err != nil {
		tbl = nil
		t.Fatalf("Failed to create empty Blacklist: %s",
			err.Error())
	}
} // func TestCreateBlacklist(t *testing.T)

func TestAddPattern(t *testing.T) {
	if tbl == nil {
		t.SkipNow()
	}

	var err error

	for _, pat := range testPatterns {
		if err = tbl.Add(pat); err != nil {
			t.Errorf("Failed to add pattern %q: %s",
				pat,
				err.Error())
		}
	}
} // func TestAddPattern(t *testing.T)
