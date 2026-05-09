// /home/krylon/go/src/github.com/blicero/newsroom/analyze/analyze.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-09 15:21:09 krylon>

// Package analyze provides analysis of the news Items.
package analyze

import (
	"embed"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
)

//go:embed corpus
var corpus embed.FS

const (
	sepPat = `[^A-Za-zÄÖÜßäöü]+`
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

type WordMap map[string]float64

// Word is a word (duh!) and the number of times it occured in a given period.
// Sorry about the name.
type Word struct {
	Word  string
	Count float64
}

func wordCmp(w1, w2 Word) int {
	if w1.Count < w2.Count {
		return 1
	} else if w1.Count > w2.Count {
		return -1
	}

	return 0
} // func wordCmp(w1, w2 Word) int

type WordList []Word

// Delta is a word and its frequency in two different time periods.
type Delta struct {
	Word  string
	Count [2]float64
}

// Change returns how much the frequency of a word has changed from one Period
// to the other.
func (d Delta) Change() float64 {
	return d.Count[1] - d.Count[0]
}

// DeltaList is a list of Delta, analogous to the WordList
type DeltaList []Delta

// TrendScout looks for the most frequent words in a given period, and how the
// distribution changed compared to earlier periods.
type TrendScout struct {
	log  *log.Logger
	sep  *regexp.Regexp
	char *regexp.Regexp
	pool *database.Pool
}

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

func (ts *TrendScout) AnalyzePeriod(p *Period, cnt int) (WordList, error) {
	var (
		err       error
		items     []*model.Item
		histogram WordMap
		db        *database.Database
		feeds     []*model.Feed
		lngMap    map[int64]string
		nameMap   map[int64]string
		hackID    int64
		tags      []*model.Tag
		tagMap    map[string]bool
	)

	db = ts.pool.Get()
	defer ts.pool.Put(db)

	if feeds, err = db.FeedGetAll(); err != nil {
		ts.log.Printf("[ERROR] Cannot load list of Feeds: %s\n",
			err.Error())
		return nil, err
	} else if items, err = db.ItemGetByPeriod(p.Begin, p.End); err != nil {
		ts.log.Printf("[ERROR] Failed to load news for Period %s to %s: %s\n",
			p.Begin.Format(common.TimestampFormatMinute),
			p.End.Format(common.TimestampFormatMinute),
			err.Error())
		return nil, err
	} else if tags, err = db.TagGetSorted(); err != nil {
		ts.log.Printf("[ERROR] Failed to load all Tags: %s\n",
			err.Error())
		return nil, err
	}

	tagMap = make(map[string]bool, len(tags))
	lngMap = make(map[int64]string, len(feeds))
	nameMap = make(map[int64]string, len(feeds))

	for _, feed := range feeds {
		lngMap[feed.ID] = feed.Language
		nameMap[feed.ID] = strings.ToLower(feed.Name)
		if feed.Name == "Hacker News" {
			hackID = feed.ID
		}
	}

	for _, tag := range tags {
		tagMap[strings.ToLower(tag.Name)] = true
	}

	histogram = make(WordMap)

	for _, item := range items {
		var content string

		if item.FeedID == hackID {
			content = item.Title
		} else {
			content = item.Strip()
		}

		var (
			words = ts.sep.Split(content, -1)
			lng   = lngMap[item.FeedID]
			score float64
			wtags []*model.Tag
			tmap  map[string]bool
		)

		if wtags, err = db.TagLinkGetByItem(item); err != nil {
			ts.log.Printf("[ERROR] Failed to load Tags for Item %q (%d): %s\n",
				item.Title,
				item.ID,
				err.Error())
			return nil, err
		}

		tmap = make(map[string]bool, len(wtags))

		for _, tag := range wtags {
			tmap[tag.Name] = true
		}

		switch item.Rating {
		case rating.Unrated:
			score = 1.0
		case rating.Boring:
			score = 0.75
		case rating.Interesting:
			score = 1.25
		}

		for _, w := range words {
			w = dict.substitute(w)
			var l = strings.ToLower(w)
			if l == nameMap[item.FeedID] {
				continue
			}

			if ts.char.MatchString(w) && !stopwords[lng][l] {
				histogram[w] += score
				if tagMap[l] && tmap[l] {
					histogram[w] += 0.25
				}
			}
		}
	}

	var list = make(WordList, 0, len(histogram))

	for w, c := range histogram {
		list = append(list, Word{Word: w, Count: c})
	}

	slices.SortFunc(list, wordCmp)

	if len(list) > cnt {
		return list[:cnt], nil
	}

	return list, nil
} // func (ts *TrendScout) AnalyzePeriod(p *Period) (WordMap, error)

// AnalyzeDelta calculates the changes in word frequencies between two Periods.
func (ts *TrendScout) AnalyzeDelta(p1, p2 *Period, cnt int) (DeltaList, error) {
	var (
		err    error
		wg     sync.WaitGroup
		errCnt atomic.Int64
		wlists [2]WordList
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		var (
			ex    error
			wlist WordList
		)

		if wlist, ex = ts.AnalyzePeriod(p1, cnt); ex != nil {
			errCnt.Add(1)
			ts.log.Printf("[ERROR] Failed to analyze Period %s: %s\n",
				p1,
				err.Error())
		} else {
			wlists[0] = wlist
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var (
			ex    error
			wlist WordList
		)

		if wlist, ex = ts.AnalyzePeriod(p2, cnt); ex != nil {
			errCnt.Add(1)
			ts.log.Printf("[ERROR] Failed to analyze Period %s: %s\n",
				p2,
				err.Error())
		} else {
			wlists[1] = wlist
		}
	}()

} // func (ts *TrendScout) AnalyzeDelta(p1, p2 *Period, cnt int) (DeltaList, error)
