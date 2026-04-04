// /home/krylon/go/src/github.com/blicero/newsroom/database/feed.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-04 18:21:09 krylon>

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/blicero/newsroom/database/query"
	"github.com/blicero/newsroom/model"
)

// FeedAdd adds a new Feed to the Database.
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
		int64(math.Floor(f.RefreshInterval.Seconds()))); err != nil {
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

// FeedGetByID loads a Feed by its ID.
func (db *Database) FeedGetByID(id int64) (*model.Feed, error) {
	const qid query.ID = query.FeedGetByID
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
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			f                     = &model.Feed{ID: id}
			interval, lastRefresh int64
			ustr, homepage        string
		)

		if err = rows.Scan(&f.Name, &f.Language, &ustr, &homepage, &interval, &lastRefresh, &f.Active); err != nil {
			var ex = fmt.Errorf("failed to scan row: %w", err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if f.URL, err = url.Parse(ustr); err != nil {
			var ex = fmt.Errorf("cannot parse URL %q: %w",
				ustr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if f.Homepage, err = url.Parse(homepage); err != nil {
			var ex = fmt.Errorf("cannot parse homepage %q: %w",
				homepage,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		f.RefreshInterval = time.Second * time.Duration(interval)
		f.LastRefresh = time.Unix(lastRefresh, 0)
		return f, nil
	}

	return nil, nil
} // func (db *Database) FeedGetByID(id int64) (*model.Feed, error)

// FeedGetDue loads all Feeds that are due for a refresh.
func (db *Database) FeedGetDue() ([]*model.Feed, error) {
	const qid query.ID = query.FeedGetDue
	var err error
	var msg string
	var stmt *sql.Stmt
	var feeds []*model.Feed

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
			msg = fmt.Sprintf("Error querying Feeds due for refresh: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	feeds = make([]*model.Feed, 0, 32)

	for rows.Next() {
		var (
			ex                error
			ustr, homepage    string
			refresh, interval int64
			feed              = new(model.Feed)
		)

		if err = rows.Scan(
			&feed.ID,
			&feed.Name,
			&feed.Language,
			&ustr,
			&homepage,
			&interval,
			&refresh,
			&feed.Active,
		); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		} else if feed.URL, err = url.Parse(ustr); err != nil {
			ex = fmt.Errorf("cannot parse URL %q: %w",
				ustr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if feed.Homepage, err = url.Parse(homepage); err != nil {
			ex = fmt.Errorf("cannot parse homepage %q: %w",
				ustr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		feed.RefreshInterval = time.Second * time.Duration(interval)
		feed.LastRefresh = time.Unix(refresh, 0)

		feeds = append(feeds, feed)
	}

	return feeds, nil
} // func (db *Database) FeedGetDue() ([]*model.Feed, error)

// FeedGetAll fetches all feeds from the database.
func (db *Database) FeedGetAll() ([]*model.Feed, error) {
	const qid query.ID = query.FeedGetAll
	var err error
	var msg string
	var stmt *sql.Stmt
	var feeds []*model.Feed

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
			msg = fmt.Sprintf("Error querying all Feeds: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close() // nolint: errcheck
	}

	feeds = make([]*model.Feed, 0, 32)

	for rows.Next() {
		var (
			ex                error
			ustr, homepage    string
			refresh, interval int64
			feed              = new(model.Feed)
		)

		if err = rows.Scan(
			&feed.ID,
			&feed.Name,
			&feed.Language,
			&ustr,
			&homepage,
			&interval,
			&refresh,
			&feed.Active,
		); err != nil {
			msg = fmt.Sprintf("error scanning row: %s", err.Error())
			db.log.Printf("[ERROR] %s\n", msg)
			return nil, errors.New(msg)
		} else if feed.URL, err = url.Parse(ustr); err != nil {
			ex = fmt.Errorf("cannot parse URL %q: %w",
				ustr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		} else if feed.Homepage, err = url.Parse(homepage); err != nil {
			ex = fmt.Errorf("cannot parse homepage %q: %w",
				ustr,
				err)
			db.log.Printf("[ERROR] %s\n", ex.Error())
			return nil, ex
		}

		feed.RefreshInterval = time.Second * time.Duration(interval)
		feed.LastRefresh = time.Unix(refresh, 0)

		feeds = append(feeds, feed)
	}

	return feeds, nil
} // func (db *Database) FeedGetAll(limit int64) ([]*model.Feed, error)

// FeedSetInterval sets a Feeds Refresh Interval.
func (db *Database) FeedSetInterval(feed *model.Feed, interval time.Duration) error {
	const qid query.ID = query.FeedSetInterval
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
	if res, err = stmt.Exec(interval.Seconds(), feed.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot update interval of Feed %s (%d): %w",
				feed.Name,
				feed.ID,
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

	feed.RefreshInterval = interval
	return nil
} // func (db *Database) FeedSetInterval(feed *model.Feed, interval time.Duration) error

// FeedSetLastRefresh sets a Feeds refresh timestamp.
func (db *Database) FeedSetLastRefresh(feed *model.Feed, when time.Time) error {
	const qid query.ID = query.FeedSetLastRefresh
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
	if res, err = stmt.Exec(when.Unix(), feed.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot update interval of Feed %s (%d): %w",
				feed.Name,
				feed.ID,
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

	feed.LastRefresh = when
	return nil
} // func (db *Database) FeedSetLastRefresh(feed *model.Feed, interval time.Duration) error

// FeedSetActive sets a Feed's paused flag.
func (db *Database) FeedSetActive(feed *model.Feed, active bool) error {
	const qid query.ID = query.FeedSetActive
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
	if res, err = stmt.Exec(active, feed.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot update paused flag of Feed %s (%d): %w",
				feed.Name,
				feed.ID,
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

	feed.Active = active
	return nil
} // func (db *Database) FeedSetPause(feed *model.Feed, paused bool) error

// FeedDelete deletes a Feed from the Database.
func (db *Database) FeedDelete(feed *model.Feed) error {
	const qid query.ID = query.FeedDelete
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
	if res, err = stmt.Exec(feed.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			ex = fmt.Errorf("cannot delete Feed %s (%d): %w",
				feed.Name,
				feed.ID,
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
} // func (db *Database) FeedDelete(feed *model.Feed) error
