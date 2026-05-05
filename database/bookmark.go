// /home/krylon/go/src/github.com/blicero/newsroom/database/later.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-05 11:22:48 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/blicero/newsroom/database/query"
	"github.com/blicero/newsroom/model"
)

// BookmarkAdd marks an Item to be read later.
func (db *Database) BookmarkAdd(bookmark *model.Bookmark) error {
	const qid query.ID = query.BookmarkAdd
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
	if rows, err = stmt.Query(bookmark.ItemID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot mark Item %d for later reading: %w",
				bookmark.ItemID,
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
			var ex = fmt.Errorf("failed to get ID for newly added Later %d: %w",
				bookmark.ItemID,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		bookmark.ID = id
		return nil
	}
} // func (db *Database) LaterAdd(later *model.Later) error

// BookmarkGetAll fetches all bookmarks from the database.
func (db *Database) BookmarkGetAll() ([]*model.Bookmark, error) {
	const qid query.ID = query.BookmarkGetAll
	var (
		err       error
		msg       string
		stmt      *sql.Stmt
		bookmarks []*model.Bookmark
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
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying all Later marks: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	bookmarks = make([]*model.Bookmark, 0, 8)

	for rows.Next() {
		var (
			bookmark           = new(model.Bookmark)
			deadline, finished int64
		)

		if err = rows.Scan(
			&bookmark.ID,
			&bookmark.ItemID,
			&deadline,
			&bookmark.Comment,
			&bookmark.Finished,
			&finished,
		); err != nil {
			var ex = fmt.Errorf("error scanning row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		if deadline != 0 {
			bookmark.Deadline = time.Unix(deadline, 0)
		}
		if finished != 0 {
			bookmark.FinishedWhen = time.Unix(finished, 0)
		}
		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
} // func (db *Database) LaterGetAll() ([]*model.Later, error)

// BookmarkDelete removes a bookmark from the Database.
func (db *Database) BookmarkDelete(later *model.Bookmark) error {
	const qid query.ID = query.BookmarkDelete
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

EXEC_QUERY:
	if _, err = stmt.Exec(later.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot delete bookmark %d: %w",
				later.ID,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}
	}

	return nil
} // func (db *Database) LaterDelete(later *model.Later) error

// BookmarkMarkFinished marks a bookmark as finished.
func (db *Database) BookmarkMarkFinished(bookmark *model.Bookmark) error {
	const qid query.ID = query.BookmarkMarkFinished
	var (
		err, ex     error
		stmt        *sql.Stmt
		finishStamp = time.Now()
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
	if _, err = stmt.Exec(finishStamp.Unix(), bookmark.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot delete bookmark %d: %w",
				bookmark.ID,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}
	}

	bookmark.Finished = true
	bookmark.FinishedWhen = finishStamp
	return nil
} // func (db *Database) LaterMarkFinished(later *model.Later) error
