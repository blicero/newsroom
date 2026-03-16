// /home/krylon/go/src/github.com/blicero/newsroom/web/ajax_types.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2026-03-16 15:33:49 krylon>

package web

import (
	"time"
)

type ajaxData struct {
	Status    bool      `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type ajaxBeaconData struct {
	ajaxData
	Hostname string `json:"hostname"`
}

type ajaxResponseRateItem struct {
	ajaxData
	Content string `json:"content"`
}
