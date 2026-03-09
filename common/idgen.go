// /home/krylon/go/src/github.com/blicero/grace/common/idgen.go
// -*- mode: go; coding: utf-8; -*-
// Created on 29. 12. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2026-03-02 17:46:40 krylon>

package common

import (
	"sync/atomic"
)

// IDGen generates unique IDs (unique per IDGenerator, that is).
type IDGen struct {
	cnt atomic.Int64
}

// Next returns a fresh, unique ID.
func (g *IDGen) Next() int64 {
	return g.cnt.Add(1)
} // func (g *IDGen) Next() int64
