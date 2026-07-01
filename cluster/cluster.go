// /home/krylon/go/src/github.com/blicero/newsroom/cluster/cluster.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 07. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-07-01 11:24:58 krylon>

// Package cluster uses Latent Semantic Analysis (LSA) to find related Items.
// At least that is the idea.
package cluster

import (
	"log"
	"time"

	"github.com/blicero/krylib"
	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
)

// When looking for related Items, processing ALL Items in the
// Database is not going to be practical.
//
// For starters, I will look into Items that were added within, say, a
// week of the one we are starting with.

const clusterPeriod = time.Hour * 24 * 7

// SemanticCluster represents an Item and a group of Items that are similar
// or related.
type SemanticCluster struct {
	Root    *model.Item
	Related []*model.Item
}

// Scout finds clusters of related Items. Hopefully.
type Scout struct {
	log *log.Logger
	db  *database.Database
}

// NewScout creates and returns a fresh Scout.
func NewScout() (*Scout, error) {
	var (
		err error
		s   = new(Scout)
	)

	if s.log, err = common.GetLogger(logdomain.Cluster); err != nil {
		return nil, err
	} else if s.db, err = database.Open(common.DbPath); err != nil {
		s.log.Printf("[CRITICAL] Cannot open Database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	}

	return s, nil
} // func NewScout() (*Scout, error)

// FindCluster attempts to find Items related to the given argument.
func (s *Scout) FindCluster(item *model.Item) (*SemanticCluster, error) {
	var (
		err        error
		begin, end time.Time
		items      []*model.Item
		clu        *SemanticCluster
	)

	begin = item.Timestamp.Add(-clusterPeriod)
	end = item.Timestamp.Add(clusterPeriod)

	if items, err = s.db.ItemGetByPeriod(begin, end); err != nil {
		s.log.Printf("[ERROR] Failed to find Items by Period [%s - %s]: %s\n",
			begin.Format(common.TimestampFormatMinute),
			end.Format(common.TimestampFormatMinute),
			err.Error())
	}

	return nil, krylib.ErrNotImplemented
} // func (s *Scout) FindCluster(item *model.Item) (*SemanticCluster, error)
