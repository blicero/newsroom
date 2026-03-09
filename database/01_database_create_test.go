// /home/krylon/go/src/github.com/blicero/guangng/database/01_database_create_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-09 14:16:19 krylon>

package database

import (
	"database/sql"
	"testing"

	"github.com/blicero/newsroom/common"
)

var tdb *Database

func TestCreateDB(t *testing.T) {
	var err error

	if tdb, err = Open(common.DbPath); err != nil {
		tdb = nil
		t.Fatalf("Cannot create database: %s",
			err.Error())
	}
} // func TestCreateDB(t *testing.T)

func TestQueryPrepare(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err error
		q   *sql.Stmt
	)

	for k, s := range qdb {
		if q, err = tdb.getQuery(k); err != nil {
			t.Errorf("Error preparing query %s: %s\n%s\n",
				k,
				err.Error(),
				s)
		} else if q == nil {
			t.Errorf("Query handle %s is nil!", k)
		}
	}
} // func TestQueryPrepare(t *testing.T)
