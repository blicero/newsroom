// /home/krylon/go/src/github.com/blicero/newsroom/database/05_tag_link_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-08 12:42:07 krylon>

package database

import (
	"math/rand"
	"testing"

	"github.com/blicero/newsroom/model"
)

const lnkCnt = itemCnt * 2

var links [lnkCnt]*model.TagLink

func TestLinkAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var lidx = 0

	for idx := range itemCnt {
		var (
			err  error
			tids = rand.Perm(len(tags))
			lnk  = &model.TagLink{
				TagID:  tags[tids[0]].ID,
				ItemID: titems[idx].ID,
			}
		)

		if err = tdb.TagLinkAdd(lnk); err != nil {
			t.Fatalf("Failed to link Tag %d to Item %d: %s",
				lnk.TagID,
				lnk.ItemID,
				err.Error())
		} else if lnk.ID == 0 {
			t.Fatalf("Adding TagLink(%d -> %d) raised no error, but it has no ID",
				lnk.TagID,
				lnk.ItemID)
		}

		links[lidx] = lnk
		lidx++
		lnk = &model.TagLink{
			TagID:  tags[tids[1]].ID,
			ItemID: titems[idx].ID,
		}

		if err = tdb.TagLinkAdd(lnk); err != nil {
			t.Fatalf("Failed to link Tag %d to Item %d: %s",
				lnk.TagID,
				lnk.ItemID,
				err.Error())
		} else if lnk.ID == 0 {
			t.Fatalf("Adding TagLink(%d -> %d) raised no error, but it has no ID",
				lnk.TagID,
				lnk.ItemID)
		}

		links[lidx] = lnk
		lidx++
	}
} // func TestLinkAdd(t *testing.T)
