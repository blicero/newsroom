// /home/krylon/go/src/github.com/blicero/newsroom/database/tag.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-07 15:43:13 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/blicero/newsroom/database/query"
	"github.com/blicero/newsroom/model"
)

// TagAdd adds a new Tag to the Database.
func (db *Database) TagAdd(tag *model.Tag) error {
	const qid query.ID = query.TagAdd
	var (
		err, ex error
		stmt    *sql.Stmt
		parent  *int64
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Failed to prepare query %s: %s\n",
			qid,
			err.Error())
		panic(err)
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	if tag.ParentID != 0 {
		parent = new(int64)
		*parent = tag.ParentID
	}

	var rows *sql.Rows
EXEC_QUERY:
	if rows, err = stmt.Query(
		tag.Name,
		parent,
	); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot add Tag %s: %w",
				tag.Name,
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
			var ex = fmt.Errorf("failed to get ID for newly added Tag %q: %w",
				tag.Name,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return ex
		}

		tag.ID = id
		return nil
	}
} // func (db *Database) TagAdd(tag *model.Tag) error

// TagGetAll loads all Tags from the Database.
func (db *Database) TagGetAll() ([]*model.Tag, error) {
	const qid query.ID = query.TagGetAll
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
			msg = fmt.Sprintf("Error querying all Tags: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	var tags = make([]*model.Tag, 0, 32)

	for rows.Next() {
		var tag = new(model.Tag)

		if err = rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.ParentID,
		); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		tags = append(tags, tag)
	}

	return tags, nil
} // func (db *Database) TagGetAll() ([]*model.Tag, error)

// TagGetSorted loads all Tags from the Database, sorting them by hierarchy first
// and by Name second.
func (db *Database) TagGetSorted() ([]*model.Tag, error) {
	const qid query.ID = query.TagGetSorted
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
			msg = fmt.Sprintf("Error querying all Tags: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	var tags = make([]*model.Tag, 0, 32)

	for rows.Next() {
		var tag = new(model.Tag)

		if err = rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.ParentID,
			&tag.Level,
			&tag.FullName,
		); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		tags = append(tags, tag)
	}

	return tags, nil
} // func (db *Database) TagGetSorted() ([]*model.Tag, error)

// TagGetByID looks up a Tag by its Database ID.
func (db *Database) TagGetByID(id int64) (*model.Tag, error) {
	const qid query.ID = query.TagGetByID
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
			return nil, err
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying all Tags: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	var tag = &model.Tag{ID: id}

	if rows.Next() {
		if err = rows.Scan(
			&tag.Name,
			&tag.ParentID,
		); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		}

		return tag, nil
	}

	db.log.Printf("[TRACE] Tag #%d was not found in Database.\n",
		id)

	return nil, nil
} // func (db *Database) TagGetByID(id int64) (*model.Tag, error)

// TagDelete removes a Tag from the Database. It is not possible to remove
// a Tag that has children.
func (db *Database) TagDelete(tag *model.Tag) error {
	const qid query.ID = query.TagDelete
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
	if res, err = stmt.Exec(tag.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot delete Tag %s (%d): %w",
				tag.Name,
				tag.ID,
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
} // func (db *Database) TagDelete(tag *model.Tag) error

// TagSetParent updates a Tag's ParentID. The new ParentID has to be either
// the ID of an existing Tag or 0.
func (db *Database) TagSetParent(tag *model.Tag, parentID int64) error {
	const qid query.ID = query.TagSetParent
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
	if res, err = stmt.Exec(parentID, tag.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot update parent of Tag %s (%d): %w",
				tag.Name,
				tag.ID,
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

	tag.ParentID = parentID
	return nil
} // func (db *Database) TagSetParent(tag *model.Tag, parentID int64) error
