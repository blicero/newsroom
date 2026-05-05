// /home/krylon/go/src/github.com/blicero/newsroom/analyze/analyze.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-05 14:10:05 krylon>

// Package analyze provides analysis of the news Items.
package analyze

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
)

const (
	sepPat = `\W+`
	char   = `^[[:alpha:]]+$`
)

// Period represents a timespan.
type Period struct {
	Begin time.Time
	End   time.Time
}

// NewPeriod creates a new Period, beginning at the specified Time, ending
// after the specified Duration.
func NewPeriod(begin time.Time, dur time.Duration) *Period {
	if dur < 0 {
		panic("Duration cannot be negative")
	}

	return &Period{
		Begin: begin,
		End:   begin.Add(dur),
	}
} // func NewPeriod(begin time.Time, dur time.Duration) *Period

// Duration returns the Period's duration.
func (p *Period) Duration() time.Duration {
	return p.End.Sub(p.Begin)
} // func (p *Period) Duration() time.Duration

func (p *Period) String() string {
	return fmt.Sprintf("Period{%s -- %s}",
		p.Begin.Format(common.TimestampFormatMinute),
		p.End.Format(common.TimestampFormatMinute))
} // func (p *Period) String() string

func (p *Period) Previous() *Period {
	return &Period{
		Begin: p.Begin.Add(-p.Duration()),
		End:   p.Begin,
	}
} // func (p *Period) Previous() *Period

func (p *Period) Next() *Period {
	return &Period{
		Begin: p.End,
		End:   p.End.Add(p.Duration()),
	}
} // func (p *Period) Next() *Period

// TrendScout looks for the most frequent words in a given period, and how the
// distribution changed compared to earlier periods.
type TrendScout struct {
	log  *log.Logger
	sep  *regexp.Regexp
	char *regexp.Regexp
	pool *database.Pool
}

type WordMap map[string]int64

// NewTrendScout creates a new TrendScout
func NewTrendScout() (*TrendScout, error) {
	var (
		err error
		ts  = new(TrendScout)
	)

	if ts.log, err = common.GetLogger(logdomain.Analyze); err != nil {
		return nil, err
	} else if ts.sep, err = regexp.Compile(sepPat); err != nil {
		ts.log.Printf("[CRITICAL] Cannot compile regex for word seprator %q: %s\n",
			sepPat,
			err.Error())
		return nil, err
	} else if ts.char, err = regexp.Compile(char); err != nil {
		ts.log.Printf("[CRITICAL] Cannot compile regex for characters %q: %s\n",
			char,
			err.Error())
		return nil, err
	} else if ts.pool, err = database.NewPool(4); err != nil {
		ts.log.Printf("[CRITICAL] Cannot create database pool: %s\n",
			err.Error())
		return nil, err
	}

	return ts, nil
} // func NewTrendScout() (*TrendScout, error)

func (ts *TrendScout) AnalyzePeriod(p *Period) (WordMap, error) {
	var (
		err       error
		items     []*model.Item
		histogram WordMap
		db        *database.Database
	)

	db = ts.pool.Get()
	defer ts.pool.Put(db)

	if items, err = db.ItemGetByPeriod(p.Begin, p.End); err != nil {
		ts.log.Printf("[ERROR] Failed to load news for Period %s to %s: %s\n",
			p.Begin.Format(common.TimestampFormatMinute),
			p.End.Format(common.TimestampFormatMinute),
			err.Error())
		return nil, err
	}

	histogram = make(WordMap)

	for _, item := range items {
		var (
			content = item.Strip()
			words   = ts.sep.Split(content, -1)
		)

		for _, w := range words {
			if !ts.char.MatchString(w) || len(w) < 2 {
				continue
			}
			histogram[w]++
		}
	}

	return histogram, nil
} // func (ts *TrendScout) AnalyzePeriod(p *Period) (WordMap, error)
