// /home/krylon/go/src/github.com/blicero/newsroom/database/search.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-21 13:00:56 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database/query"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
	"github.com/jmoiron/sqlx"
)

// SearchParms describes a search to be performed on the news Items.
type SearchParms struct {
	DateP     bool
	DateRange [2]time.Time
	TagP      bool
	Tags      map[int64]bool
	Query     string
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
		tags = make([]int64, len(parm.Tags))
		for tid := range parm.Tags {
			tags[tidx] = tid
			tidx++
		}

		switch qid {
		case query.ItemSearchTag:
			qstr, qargs, err = sqlx.In(
				qdb[qid],
				tags,
				parm.Query, parm.Query)
		case query.ItemSearchDateTag:
			qstr, qargs, err = sqlx.In(
				qdb[qid],
				parm.DateRange[0].Unix(),
				parm.DateRange[1].Unix(),
				tags,
				parm.Query, parm.Query)
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
		rows, err = stmt.Query(parm.Query, parm.Query)
	case query.ItemSearchDate:
		rows, err = stmt.Query(
			parm.DateRange[0].Unix(),
			parm.DateRange[1].Unix(),
			parm.Query,
			parm.Query)
	case query.ItemSearchTag:
		tags = make([]int64, len(parm.Tags))
		for id := range parm.Tags {
			tags[tidx] = id
			tidx++
		}

		rows, err = stmt.Query(
			tags,
			parm.Query,
			parm.Query)
	case query.ItemSearchDateTag:
		tags = make([]int64, len(parm.Tags))
		for id := range parm.Tags {
			tags[tidx] = id
			tidx++
		}

		rows, err = stmt.Query(
			parm.DateRange[0].Unix(),
			parm.DateRange[1].Unix(),
			tags,
			parm.Query,
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

	return items, nil
} // func (db *Database) Search(parm *SearchParms) ([]*model.Item, error)
