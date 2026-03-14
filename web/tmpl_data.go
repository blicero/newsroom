// /home/krylon/go/src/newsroom/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2026-03-14 12:42:11 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

import "github.com/blicero/newsroom/model"

type tmplDataBase struct {
	Title string
	Debug bool
	URL   string
}

type tmplDataIndex struct {
	tmplDataBase
	Feeds []*model.Feed
}

type tmplDataNews struct {
	tmplDataBase
	PageNo     int64
	Count      int64
	TotalCount int64
	MaxPage    int64
	Feeds      map[int64]*model.Feed
	Items      []*model.Item
}

// FirstPage returns true if we are on the first page.
func (tdn *tmplDataNews) FirstPage() bool {
	return tdn.PageNo == 0
} // func (tdn *tmplDataNews) FirstPage() bool

// LastPage returns true if we are on the last page.
func (tdn *tmplDataNews) LastPage() bool {
	return tdn.PageNo >= tdn.MaxPage
} // func (tdn *tmplDataNews) LastPage() bool
