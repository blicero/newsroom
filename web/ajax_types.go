// /home/krylon/go/src/github.com/blicero/newsroom/web/ajax_types.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2026-03-14 13:10:01 krylon>

package web

import (
	"time"
)

type ajaxData struct {
	Status    bool
	Message   string
	Timestamp time.Time
}

type ajaxBeaconData struct {
	ajaxData
	Hostname string
}

type ajaxResponseRateItem struct {
	ajaxData
}
