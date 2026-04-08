// /home/krylon/go/src/github.com/blicero/newsroom/database/03_tag_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-02 14:38:08 krylon>

package database

import (
	"testing"

	"github.com/blicero/newsroom/model"
)

var tags = []*model.Tag{
	{Name: "Culture"},
	{Name: "IT"},
	{Name: "Politics"},
	{Name: "Nature"},
}

func TestTagAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	for _, tag := range tags {
		var err error

		if err = tdb.TagAdd(tag); err != nil {
			t.Errorf("Failed to add Tag %s: %s",
				tag.Name,
				err.Error())
		} else if tag.ID == 0 {
			t.Errorf("After adding Tag %s, its ID is still 0",
				tag.Name)
		}
	}
} // func TestTagAdd(t *testing.T)

func TestTagGetAll(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err   error
		dtags []*model.Tag
	)

	if dtags, err = tdb.TagGetAll(); err != nil {
		t.Errorf("Failed to load all Tags: %s",
			err.Error())
	} else if len(dtags) != len(tags) {
		t.Errorf("Unexpected number of Tags returned from Database: %d (should be %d)",
			len(dtags),
			len(tags))
	}
} // func TestTagGetAll(t *testing.T)
