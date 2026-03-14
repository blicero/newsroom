// /home/krylon/go/src/newsroom/web/helpers_web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 09. 2019 by Benjamin Walkenhorst
// (c) 2019 Benjamin Walkenhorst
// Time-stamp: <2026-03-14 13:08:32 krylon>
//
// Helper functions for use by the HTTP request handlers

package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/blicero/newsroom/common"
)

func errJSON(msg string) []byte {
	var res = fmt.Sprintf(`{ "Status": false, "Message": %q }`,
		jsonEscape(msg))

	return []byte(res)
} // func errJSON(msg string) []byte

func jsonEscape(i string) string { // nolint: unused
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	// Trim the beginning and trailing " character
	return string(b[1 : len(b)-1])
}

func (srv *Server) baseData(title string, r *http.Request) tmplDataBase { // nolint: unused
	return tmplDataBase{
		Title: title,
		Debug: common.Debug,
		URL:   r.URL.String(),
	}
} // func (srv *Server) baseData(title string, r *http.Request) tmplDataBase
