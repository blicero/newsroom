// /home/krylon/go/src/github.com/blicero/newsroom/blacklist/blacklist.go
// -*- mode: go; coding: utf-8; -*-
// Created on 13. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-14 14:14:48 krylon>

// Package blacklist provides a filter made of one or more regular expressions,
// to exclude or hide news Items.
package blacklist

import (
	"cmp"
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"go.etcd.io/bbolt"
)

var bucketName = []byte("blacklist")

type blPat struct {
	Pattern *regexp.Regexp
	HitCnt  atomic.Int64
}

func (p *blPat) key() []byte {
	return []byte(p.Pattern.String())
} // func (p *blPat) Key() []byte

func (p *blPat) val() []byte {
	return []byte(strconv.FormatInt(p.HitCnt.Load(), 10))
} // func (p *blPat) Val() []byte

// Match returns true if the Pattern matches the Item's Title.
// In case of a match, the Patterns HitCnt is incremented as well.
func (p *blPat) Match(item *model.Item) bool {
	if p.Pattern.MatchString(item.Title) {
		p.HitCnt.Add(1)
		return true
	}

	return false
} // func (p *blPat) Match(item *model.Item) bool

func cmpPat(a, b *blPat) int {
	return -cmp.Compare(a.HitCnt.Load(), b.HitCnt.Load())
} // func cmpPat(a, b *blPat) int

// DispPat is for external use, i.e. for displaying Patterns in the web frontend.
type DispPat struct {
	Pattern string
	HitCnt  int64
}

// Blacklist matches news Items against a list of regular expressions.
type Blacklist struct {
	log      *log.Logger
	lock     sync.RWMutex
	patterns []*blPat
	db       *bbolt.DB
}

// New creates a new Blacklist.
func New() (*Blacklist, error) {
	var (
		err error
		bl  = new(Blacklist)
		opt = bbolt.Options{
			Timeout: time.Millisecond * 500,
		}
	)

	if bl.log, err = common.GetLogger(logdomain.Blacklist); err != nil {
		return nil, err
	} else if bl.db, err = bbolt.Open(common.BlacklistPath, fs.ModePerm, &opt); err != nil {
		bl.log.Printf("[CRITICAL] Failed to open Blacklist DB at %s: %s\n",
			common.BlacklistPath,
			err.Error())
		return nil, err
	} else if err = bl.initBlacklist(); err != nil {
		return nil, err
	}

	return bl, nil
} // func New() (*Blacklist, error)

func (bl *Blacklist) initBlacklist() error {
	var (
		err      error
		patterns = make([]*blPat, 0)
	)

	err = bl.db.Update(func(tx *bbolt.Tx) error {
		var (
			ex     error
			bucket *bbolt.Bucket
			cur    *bbolt.Cursor
		)

		// ...
		if bucket, ex = tx.CreateBucketIfNotExists(bucketName); ex != nil {
			bl.log.Printf("[ERROR] Cannot create/obtain Bucket %s: %s\n",
				bucketName,
				ex.Error())
			return ex
		}

		cur = bucket.Cursor()

		for key, val := cur.First(); key != nil; key, val = cur.Next() {
			var (
				patTxt, cntStr string
				cnt            int64
				pat            = new(blPat)
			)

			patTxt = string(key)
			cntStr = string(val)

			if pat.Pattern, ex = regexp.Compile(patTxt); ex != nil {
				bl.log.Printf("[ERROR] Failed to compile regex %q: %s\n",
					patTxt,
					err.Error())
				return ex
			} else if cnt, ex = strconv.ParseInt(cntStr, 10, 64); ex != nil {
				bl.log.Printf("[ERROR] Failed to parse hit count for pattern %q (%s): %s\n",
					patTxt,
					cntStr,
					err.Error())
				return ex
			}

			pat.HitCnt.Add(cnt)
			patterns = append(patterns, pat)
		}

		return ex
	})

	if err != nil {
		bl.log.Printf("[CRITICAL] Failed load Blacklist: %s\n",
			err.Error())
	} else {
		bl.patterns = patterns
		slices.SortFunc(bl.patterns, cmpPat)
	}

	return err
} // func (bl *Blacklist) initBlacklist() error

// Patterns returns a slice of DispPats representing the Blacklist for use in
// web UI.
func (bl *Blacklist) Patterns() []DispPat {
	bl.lock.RLock()
	defer bl.lock.RUnlock()

	var lst = make([]DispPat, len(bl.patterns))

	for idx, pat := range bl.patterns {
		lst[idx] = DispPat{
			Pattern: pat.Pattern.String(),
			HitCnt:  pat.HitCnt.Load(),
		}
	}

	return lst
} // func (bl *Blacklist) Patterns() []DispPat

// Add adds a new pattern to the Blacklist.
func (bl *Blacklist) Add(pattern string) error {
	var (
		err error
		pat = new(blPat)
	)

	if pat.Pattern, err = regexp.Compile(pattern); err != nil {
		bl.log.Printf("[ERROR] Cannot compile new Blacklist pattern %q: %s\n",
			pattern,
			err.Error())
		return err
	}

	bl.lock.Lock()
	defer bl.lock.Unlock()

	if err = bl.db.Update(func(tx *bbolt.Tx) error {
		var (
			ex             error
			bucket         *bbolt.Bucket
			keybuf, valbuf []byte
		)

		keybuf = []byte(pattern)
		valbuf = []byte("0")

		if bucket, ex = tx.CreateBucketIfNotExists(bucketName); ex != nil {
			bl.log.Printf("[ERROR] Failed to create/obtain bucket: %s\n",
				ex.Error())
			return ex
		} else if ex = bucket.Put(keybuf, valbuf); err != nil {
			bl.log.Printf("[ERROR] Failed to store new Blacklist pattern %q: %s\n",
				pattern,
				ex.Error())
			return ex
		}

		return nil
	}); err != nil {
		bl.log.Printf("[ERROR] Failed to store new Blacklist pattern %q: %s\n",
			pattern,
			err.Error())
		return err
	}

	bl.patterns = append(bl.patterns, pat)

	return nil
} // func (bl *Blacklist) Add(pattern string) error

// Remove removes a pattern from the Blacklist.
func (bl *Blacklist) Remove(pattern string) error {
	var (
		err    error
		patIdx = -1
	)

	bl.lock.Lock()
	defer bl.lock.Unlock()

	for idx, pat := range bl.patterns {
		if pat.Pattern.String() == pattern {
			// bl.patterns = slices.Delete(bl.patterns, idx, idx)
			patIdx = idx
			break
		}
	}

	if patIdx == -1 {
		err = fmt.Errorf("pattern %q was not found in Blacklist",
			pattern)
		bl.log.Printf("[ERROR] %s\n", err.Error())
		return err
	}

	var keybuf = []byte(pattern)

	if err = bl.db.Update(func(tx *bbolt.Tx) error {
		var (
			ex     error
			bucket *bbolt.Bucket
		)

		if bucket, ex = tx.CreateBucketIfNotExists(bucketName); ex != nil {
			bl.log.Printf("[ERROR] Could not obtain/create bucket: %s\n",
				ex.Error())
			return ex
		} else if ex = bucket.Delete(keybuf); ex != nil {
			bl.log.Printf("[ERROR] Failed to delete Blacklist pattern %q: %s\n",
				pattern,
				ex.Error())
			return ex
		}

		return nil
	}); err != nil {
		bl.log.Printf("[ERROR] Failed to remove pattern %q: %s\n",
			pattern,
			err.Error())
		return err
	}

	bl.patterns = slices.Delete(bl.patterns, patIdx, patIdx+1)
	return nil
} // func (bl *Blacklist) Remove(pattern string) error

// Sort rearranges patterns in the Blacklist so that Pattern with higher hit
// counts move towards the front.
func (bl *Blacklist) Sort() {
	bl.lock.Lock()
	slices.SortFunc(bl.patterns, cmpPat)
	bl.lock.Unlock()
} // func (bl *Blacklist) Sort()

// Save persists the Blacklist's patterns and their counters.
func (bl *Blacklist) Save() error {
	bl.lock.Lock()
	defer bl.lock.Unlock()

	bl.log.Println("[TRACE] Saving Blacklist.")

	var err error

	if err = bl.db.Update(func(tx *bbolt.Tx) error {
		var (
			ex             error
			bucket         *bbolt.Bucket
			keybuf, valbuf []byte
		)

		if bucket, ex = tx.CreateBucketIfNotExists(bucketName); ex != nil {
			bl.log.Printf("[ERROR] Failed to create/obtain bucket: %s\n",
				ex.Error())
			return ex
		}

		for _, pat := range bl.patterns {
			keybuf = pat.key()
			valbuf = pat.val()

			if ex = bucket.Put(keybuf, valbuf); err != nil {
				bl.log.Printf("[ERROR] Failed to save Blacklist pattern %q: %s\n",
					pat.Pattern.String(),
					ex.Error())
				return ex
			}
		}

		return nil
	}); err != nil {
		bl.log.Printf("[ERROR] Failed to save Blacklist: %s\n",
			err.Error())
	}

	return err
} // func (bl *Blacklist) Save() error

// Match attempts to match the Item's Headline against each pattern in the
// Blacklist, until either a match is found or the list is exhausted.
func (bl *Blacklist) Match(item *model.Item) bool {
	bl.lock.RLock()
	defer bl.lock.RUnlock()

	for _, pat := range bl.patterns {
		if pat.Match(item) {
			bl.log.Printf("[DEBUG] Pattern %q matches headline %q\n",
				pat.Pattern.String(),
				item.Title)
			return true
		}
	}

	return false
} // func (bl *Blacklist) Match(item *model.Item) bool
