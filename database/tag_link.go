// /home/krylon/go/src/github.com/blicero/newsroom/database/tag_link.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-09 13:53:00 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/blicero/newsroom/database/query"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
)

// TagLinkAdd attaches a Tag to an Item.
func (db *Database) TagLinkAdd(lnk *model.TagLink) error {
	const qid query.ID = query.TagLinkAdd
	var (
		err, ex error
		stmt    *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows
EXEC_QUERY:
	if rows, err = stmt.Query(lnk.TagID, lnk.ItemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot add Tag %d to Item %d: %w",
				lnk.TagID,
				lnk.ItemID,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}
	} else {
		var id int64

		defer rows.Close() // nolint: errcheck

		if !rows.Next() {
			// CANTHAPPEN
			db.log.Printf("[ERROR] Query %s did not return a value\n",
				qid)
			return fmt.Errorf("query %s did not return a value", qid)
		} else if err = rows.Scan(&id); err != nil {
			var ex = fmt.Errorf("failed to get ID for newly added TagLink (%d -> %d): %w",
				lnk.TagID,
				lnk.ItemID,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		lnk.ID = id
		return nil
	}
} // func (db *Database) TagLinkAdd(lnk *model.TagLink) error

// TagLinkGetByItem loads all Tags that are attached to the given Item.
func (db *Database) TagLinkGetByItem(item *model.Item) ([]*model.Tag, error) {
	const qid query.ID = query.TagLinkGetByItem
	var (
		err  error
		msg  string
		stmt *sql.Stmt
	)

GET_QUERY:
	if stmt, err = db.getQuery(qid); err != nil {
		if worthARetry(err) {
			waitForRetry()
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

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(item.ID); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying all Tags: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	}

	defer rows.Close() // nolint: errcheck

	var tags = make([]*model.Tag, 0, 8)

	for rows.Next() {
		var tag = new(model.Tag)

		if err = rows.Scan(
			&tag.ID,
			&tag.ParentID,
			&tag.Name,
		); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		tags = append(tags, tag)
	}

	return tags, nil
} // func (db *Database) TagLinkGetByItem(item *model.Item) ([]*model.Tag, error)

// TagLinkGetByTag loads all Items the given Tag is attached to.
func (db *Database) TagLinkGetByTag(tag *model.Tag) ([]*model.Item, error) {
	const qid query.ID = query.TagLinkGetByTag
	var (
		err   error
		msg   string
		stmt  *sql.Stmt
		items []*model.Item
	)

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

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(tag.ID); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying all Feeds: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	items = make([]*model.Item, 0, 32)

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
} // func (db *Database) TagLinkGetByTag(tag *model.Tag) ([]*model.Item, error)

// TagLinkDelete removes the link between a given Tag and a given Item.
func (db *Database) TagLinkDelete(tagID, itemID int64) error {
	const qid query.ID = query.TagLinkDelete
	var (
		err, ex error
		stmt    *sql.Stmt
		res     sql.Result
		cnt     int64
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if res, err = stmt.Exec(tagID, itemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot detach Tag %d from Item %d: %w",
				tagID,
				itemID,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}
	} else if cnt, err = res.RowsAffected(); err != nil {
		ex = fmt.Errorf("failed to get number of affected rows: %w",
			err)
		db.log.Printf("[ERROR] %s\n", ex.Error())
		return ex
	} else if cnt != 1 {
		ex = fmt.Errorf("unexpected number of affected rows for %s: %d (expected 1)",
			qid,
			cnt)
		db.log.Printf("[CRITICAL] %s\n", ex.Error())
		return ex
	}

	return nil
} // func (db *Database) TagLinkDelete(tag *model.Tag, item *model.Item) error
