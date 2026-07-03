// /home/krylon/go/src/newsroom/web/tmpl_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 05. 2020 by Benjamin Walkenhorst
// (c) 2020 Benjamin Walkenhorst
// Time-stamp: <2026-07-02 13:12:56 krylon>
//
// This file contains data structures to be passed to HTML templates.

package web

import (
	"time"

	"github.com/blicero/newsroom/analyze"
	"github.com/blicero/newsroom/blacklist"
	"github.com/blicero/newsroom/classify"
	"github.com/blicero/newsroom/cluster"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/model"
)

type tmplDataBase struct {
	Title      string
	Debug      bool
	URL        string
	Messages   []string
	DoRefresh  bool
	HideBoring bool
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
	Tags       []*model.Tag
	TagMap     map[int64]*model.Tag
	ItemTags   map[int64]map[int64]bool
	TagAdvice  map[int64]classify.SuggList
	Bookmarks  map[int64]*model.Bookmark
}

// FirstPage returns true if we are on the first page.
func (tdn *tmplDataNews) FirstPage() bool {
	return tdn.PageNo == 0
} // func (tdn *tmplDataNews) FirstPage() bool

// LastPage returns true if we are on the last page.
func (tdn *tmplDataNews) LastPage() bool {
	return tdn.PageNo >= tdn.MaxPage
} // func (tdn *tmplDataNews) LastPage() bool

type tmplDataRelated struct {
	tmplDataBase
	Count      int64
	TotalCount int64
	Feeds      map[int64]*model.Feed
	Items      *cluster.SemanticCluster
	Tags       []*model.Tag
	TagMap     map[int64]*model.Tag
	ItemTags   map[int64]map[int64]bool
	Bookmarks  map[int64]*model.Bookmark
}

type tmplDataTags struct {
	tmplDataBase
	Tag  *model.Tag
	Tags []*model.Tag
}

type tmplDataTagView struct {
	tmplDataNews
	Tag *model.Tag
}

type tmplDataBlacklist struct {
	tmplDataBase
	Patterns []blacklist.DispPat
}

type tmplDataSearch struct {
	tmplDataNews
	Parm     database.SearchParms
	IsResult bool
}

type tmplDataBookmarks struct {
	tmplDataBase
	Bookmarks []*model.Bookmark
	Items     map[int64]*model.Item
}

type tmplDataHistogram struct {
	tmplDataBase
	Period analyze.Period
	Words  analyze.WordList
	Tags   map[string]*model.Tag
}

type tmplDataDelta struct {
	tmplDataBase
	Period [2]analyze.Period
	Words  analyze.DeltaList
	Tags   map[string]*model.Tag
}

type tmplDataTrend struct {
	tmplDataBase
	Interval  time.Duration
	WordCount int64
	ICnt      int64
	Series    *analyze.Series
	Tags      map[string]*model.Tag
}

// Days returns the Interval in days.
func (dt *tmplDataTrend) Days() int64 {
	return int64(dt.Interval.Hours()) / 24
} // func (dt *tmplDataTrend) Days() int64

type frequency[T any] struct {
	Val T
	Cnt int64
}

func (f *frequency[T]) cmp(g *frequency[T]) int {
	if f.Cnt < g.Cnt {
		return -1
	} else if f.Cnt > g.Cnt {
		return 1
	}

	return 0
} // func (f *frequency[T]) cmp(g *frequency[T]) int

type tmplDataTagsByPeriod struct {
	tmplDataBase
	Period      analyze.Period
	Tags        []*model.Tag
	TagMap      map[int64]*model.Tag
	Frequencies []frequency[*model.Tag]
}

func (tp *tmplDataTagsByPeriod) TagCnt() int {
	return len(tp.Frequencies)
} // func (tp *tmplDataTagsByPeriod) TagCnt() int

func (tp *tmplDataTagsByPeriod) TotalLinkCnt() int {
	var cnt int64

	for _, f := range tp.Frequencies {
		cnt += f.Cnt
	}

	return int(cnt)
} // func (tp *tmplDataTagsByPeriod) TotalLinkCnt() int
