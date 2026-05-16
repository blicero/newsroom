// /home/krylon/go/src/github.com/blicero/newsroom/database/search.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-02 13:00:54 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database/query"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
	"github.com/jmoiron/sqlx"
)

// SearchParms describes a search to be performed on the news Items.
type SearchParms struct {
	DateP  bool
	Period [2]time.Time
	TagP   bool
	Tags   map[int64]bool
	Query  string
}

// Search performs a search on the database, returning Items that match
// the search parameters.
func (db *Database) Search(parm *SearchParms) ([]*model.Item, error) { // nolint: rowserr
	var (
		err       error
		qid       query.ID
		msg, qstr string
		stmt      *sql.Stmt
		items     []*model.Item
		qargs     []any
		rows      *sql.Rows
		tags      []int64
		tidx      int
	)

	db.log.Printf("[DEBUG] Searching for %#v\n", parm)

	if !strings.HasPrefix(parm.Query, "%") && !strings.HasSuffix(parm.Query, "%") {
		parm.Query = "%" + parm.Query + "%"
	} else if parm.Query == "" {
		parm.Query = "%"
	}

	if parm.DateP && parm.TagP {
		qid = query.ItemSearchDateTag
	} else if parm.DateP {
		qid = query.ItemSearchDate
	} else if parm.TagP {
		qid = query.ItemSearchTag
	} else {
		qid = query.ItemSearchPlain
	}

	if parm.TagP {
		tags = make([]int64, 0)
		for tid := range parm.Tags {
			var tlist []int64

			if tlist, err = db.expandTagHierarchy(tid); err != nil {
				db.log.Printf("[ERROR] Failed to load children of Tag %d: %s\n",
					tid,
					err.Error())
				return nil, err
			}

			tags = append(tags, tlist...)
		}

		slices.Sort(tags)

		switch qid {
		case query.ItemSearchTag:
			qstr, qargs, err = sqlx.In(
				qdb[qid],
				tags,
				parm.Query)
		case query.ItemSearchDateTag:
			qstr, qargs, err = sqlx.In(
				qdb[qid],
				parm.Period[0].Unix(),
				parm.Period[1].Unix(),
				tags,
				parm.Query)
		}

		if err != nil {
			msg = fmt.Sprintf("Failed to transform SQL query for search: %s",
				err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, err
		} else if common.Debug {
			db.log.Printf("[DEBUG] Modified SQL for search by Tags:\n\t%s\n",
				qstr)
		}

	EXEC_TAG_QUERY:
		if rows, err = db.db.Query(qstr, qargs...); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto EXEC_TAG_QUERY
			}
			db.log.Printf("[ERROR] Failed Search %s: %s\n",
				qid,
				err.Error())
			return nil, err
		}

		goto PROCESS // This is nasty!!!!
	}

GET_QUERY:
	if stmt, err = db.getQuery(qid); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			db.log.Printf("[ERROR] Error getting query %s: %s",
				qid,
				err.Error())
			return nil, err
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	switch qid {
	case query.ItemSearchPlain:
		rows, err = stmt.Query(parm.Query)
	case query.ItemSearchDate:
		rows, err = stmt.Query(
			parm.Period[0].Unix(),
			parm.Period[1].Unix(),
			parm.Query)
	case query.ItemSearchTag:
		tags = make([]int64, len(parm.Tags))
		for id := range parm.Tags {
			tags[tidx] = id
			tidx++
		}

		rows, err = stmt.Query(
			tags,
			parm.Query)
	case query.ItemSearchDateTag:
		tags = make([]int64, len(parm.Tags))
		for id := range parm.Tags {
			tags[tidx] = id
			tidx++
		}

		rows, err = stmt.Query(
			parm.Period[0].Unix(),
			parm.Period[1].Unix(),
			tags,
			parm.Query)
	}

	if err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error performing %s: %s",
				qid,
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	}

PROCESS:
	defer rows.Close() // nolint: errcheck

	items = make([]*model.Item, 0)

	for rows.Next() {
		var (
			ex                 error
			ustr               string
			timestamp, irating int64
			item               = new(model.Item)
		)

		if err = rows.Scan(
			&item.ID,
			&item.FeedID,
			&item.Title,
			&ustr,
			&irating,
			&timestamp,
			&item.Body,
		); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		} else if item.URL, err = url.Parse(ustr); err != nil {
			ex = fmt.Errorf("cannot parse URL %q: %w",
				ustr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		item.Rating = rating.Rating(irating)
		item.Timestamp = time.Unix(timestamp, 0)
		items = append(items, item)
	}

	db.log.Printf("[DEBUG] Search for %q yielded %d results\n",
		parm.Query,
		len(items))

	return items, nil
} // func (db *Database) Search(parm *SearchParms) ([]*model.Item, error)

func (db *Database) expandTagHierarchy(tagID int64) ([]int64, error) {
	var (
		err    error
		idlist []int64
		root   *model.Tag
		tags   []*model.Tag
	)

	// This a little brute force-ish, but I don't want to break my brain
	// finding a more elegant solution right now. If it turns out to kill
	// performance, I might have to, but we'll cross that bridge when
	// we get there.

	if root, err = db.TagGetByID(tagID); err != nil {
		db.log.Printf("[ERROR] Failed to load Tag %d: %s\n",
			tagID,
			err.Error())
		return nil, err
	} else if root == nil {
		err = fmt.Errorf("Tag #%d was not found in Database", tagID)
		db.log.Printf("[ERROR] %s\n",
			err.Error())
		return nil, err
	} else if tags, err = db.TagGetSorted(); err != nil {
		db.log.Printf("[ERROR] Failed to get Tag hierarchy: %s\n",
			err.Error())
		return nil, err
	}

	idlist = make([]int64, 1)
	idlist[0] = tagID

	for _, tag := range tags {
		if tag.ID != tagID && strings.HasPrefix(tag.FullName, root.FullName) {
			idlist = append(idlist, tag.ID)
		}
	}

	return idlist, nil
} // func (db *Database) expandTagHierarchy(tagID int64) ([]int64, error)
