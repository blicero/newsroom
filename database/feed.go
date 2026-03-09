// /home/krylon/go/src/github.com/blicero/newsroom/database/feed.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-09 15:18:00 krylon>

package database

import (
	"database/sql"
	"fmt"

	"github.com/blicero/newsroom/database/query"
	"github.com/blicero/newsroom/model"
)

func (db *Database) FeedAdd(f *model.Feed) error {
	const qid query.ID = query.FeedAdd
	var (
		err  error
		stmt *sql.Stmt
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
		f.Name,
		f.Language,
		f.URL.String(),
		f.Homepage.String(),
		f.RefreshInterval.Seconds()); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("cannot add Feed %s (%s): %w",
				f.Name,
				f.URL,
				err)
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
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
			var ex = fmt.Errorf("failed to get ID for newly added Feed %s: %w",
				f.Name,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		f.ID = id
		return nil
	}
} // func (db *Database) FeedAdd(f *model.Feed) error
