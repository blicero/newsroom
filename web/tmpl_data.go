// /home/krylon/go/src/newsroom/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2026-03-12 17:04:53 krylon>
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
	PageNo int64
	Count  int64
	Feeds  map[int64]*model.Feed
	Items  []*model.Item
}
