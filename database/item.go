// /home/krylon/go/src/github.com/blicero/newsroom/database/item.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-14 12:44:51 krylon>

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

// ItemAdd adds a news Item to the Database.
func (db *Database) ItemAdd(item *model.Item) error {
	const qid query.ID = query.ItemAdd
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
	if rows, err = stmt.Query(
		item.FeedID,
		item.Title,
		item.URL.String(),
		item.Timestamp.Unix(),
		item.Body,
	); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot add Item %s (%s): %w",
				item.Title,
				item.URL,
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
			var ex = fmt.Errorf("failed to get ID for newly added Item %q: %w",
				item.Title,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		item.ID = id
		return nil
	}
} // func (db *Database) ItemAdd(item *model.Item) error

// ItemGetByURL looks up an Item by its URL.
func (db *Database) ItemGetByURL(u *url.URL) (*model.Item, error) {
	const qid query.ID = query.ItemGetByURL
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(u.String()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			timestamp, irating int64
			item               = &model.Item{
				URL: u,
			}
		)

		if err = rows.Scan(&item.ID, &item.FeedID, &item.Title, &irating, &timestamp, &item.Body); err != nil {
			var ex = fmt.Errorf("failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		item.Timestamp = time.Unix(timestamp, 0)
		item.Rating = rating.Rating(irating)

		return item, nil
	}

	return nil, nil
} // func (db *Database) ItemGetByURL(u *url.URL) (*model.Item, error)

// ItemGetAll loads up to <limit> news Items from the Database, in reverse order
// of age, skipping the first <offset> Items.
// To get *ALL* Items (careful there!), pass a limit of -1.
func (db *Database) ItemGetAll(limit, offset int64) ([]*model.Item, error) {
	const qid query.ID = query.ItemGetAll
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
	if rows, err = stmt.Query(limit, offset); err != nil {
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

	items = make([]*model.Item, 0, limit)

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
} // func (db *Database) ItemGetAll(limit int64) ([]*model.Item, error)

// ItemGetByFeed loads all Items that belong to the specified Feed.
func (db *Database) ItemGetByFeed(feed *model.Feed) ([]*model.Item, error) {
	const qid query.ID = query.ItemGetByFeed
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
	if rows, err = stmt.Query(feed.ID); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying Items for Feed %s: %s",
				feed.Name,
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
			item               = &model.Item{FeedID: feed.ID}
		)

		if err = rows.Scan(
			&item.ID,
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
} // func (db *Database) ItemGetByFeed(feed *model.Feed) ([]*model.Item, error)

// ItemCount returns the total number of Items in the Database.
func (db *Database) ItemCount() (int64, error) {
	const qid query.ID = query.ItemCount
	var (
		err  error
		msg  string
		stmt *sql.Stmt
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
			return 0, err
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying total number of Items: %s",
				err.Error())
			db.log.Println(msg)
			return 0, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	if rows.Next() {
		var cnt int64

		if err = rows.Scan(&cnt); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return 0, errors.New(msg)
		}

		return cnt, nil
	}

	err = fmt.Errorf("query %s did not return a value", qid)
	db.log.Printf("[CANTHAPPEN] %s\n", err.Error())
	return 0, err
} // func (db *Database) ItemCount() (int64, error)
