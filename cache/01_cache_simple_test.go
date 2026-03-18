// /home/krylon/go/src/github.com/blicero/newsroom/cache/01_cache_simple_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-18 13:46:12 krylon>

package cache

import (
	"fmt"
	"testing"

	"github.com/blicero/newsroom/common"
)

const (
	cName = "test"
	iCnt  = 64
)

var (
	tc     *Cache[item]
	titems [iCnt]*item
)

func TestCreate(t *testing.T) {
	var (
		err error
	)

	if tc, err = New[item](cName); err != nil {
		tc = nil
		t.Fatalf("Failed to create Cache: %s",
			err.Error())
	}
} // func TestCreate(t *testing.T)

func TestStore(t *testing.T) {
	if tc == nil {
		t.SkipNow()
	}

	for i := range iCnt {
		var iname = fmt.Sprintf("item%03d", i+1)
		titems[i] = newItem(iname)
	}

	for _, it := range titems {
		if err := tc.Store(it.Name, it); err != nil {
			t.Fatalf("Failed to store test item: %s",
				err.Error())
		}
	}
} // func TestStore(t *testing.T)

func TestLoad(t *testing.T) {
	if tc == nil {
		t.SkipNow()
	}

	for _, it := range titems {
		var (
			err error
			val *item
		)

		if val, err = tc.Load(it.Name); err != nil {
			t.Fatalf("Failed to load Item %s from cache: %s",
				it.Name,
				err.Error())
		} else if val == nil {
			t.Fatalf("Item %s was not found in cache",
				it.Name)
		} else if val.ID != it.ID {
			t.Fatalf("Unexpected Item ID: %d (expected %d)",
				val.ID,
				it.ID)
		} else if val.Name != it.Name {
			t.Fatalf("Unexpected Item Name: %s (expect %s)",
				val.Name,
				it.Name)
		} else if !val.Timestamp.Equal(it.Timestamp) {
			t.Fatalf("Unexpected Item Timestamp: %s (expect %s)",
				val.Timestamp.Format(common.TimestampFormatSubSecond),
				it.Timestamp.Format(common.TimestampFormatSubSecond))
		}
	}
} // func TestLoad(t *testing.T)
