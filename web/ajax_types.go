// /home/krylon/go/src/github.com/blicero/newsroom/web/ajax_types.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2026-04-08 14:39:55 krylon>

package web

import (
	"time"

	"github.com/blicero/newsroom/model"
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

type ajaxResponseTagSubmit struct {
	ajaxData
	Operation string `json:"operation"`
	Payload   string `json:"payload"`
}

type ajaxResponseTagLinkCreate struct {
	ajaxData
	Tag *model.Tag `json:"tag"`
}
