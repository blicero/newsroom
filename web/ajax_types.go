// /home/krylon/go/src/github.com/blicero/newsroom/web/ajax_types.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2026-03-12 14:25:28 krylon>

package web

import (
	"time"
)

type ajaxData struct {
	Status    bool
	Message   string
	Timestamp time.Time
}

type ajaxCtlResponse struct {
	ajaxData
	NewCnt int
}

type ajaxWorkerCnt struct {
	ajaxData
	GeneratorAddress int
	GeneratorName    int
	XFR              int
	Scanner          int
}
