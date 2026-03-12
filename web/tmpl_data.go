// /home/krylon/go/src/newsroom/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2026-03-12 14:26:44 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

type tmplDataBase struct { // nolint: unused
	Title string
	Debug bool
	URL   string
}

type tmplDataIndex struct { // nolint: unused,deadcode
	tmplDataBase
}
