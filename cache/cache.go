// /home/krylon/go/src/github.com/blicero/newsroom/cache/cache.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-18 13:21:03 krylon>

package cache

import (
	"encoding/json"
	"io/fs"
	"log"
	"path/filepath"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/logdomain"
	"go.etcd.io/bbolt"
)

const timeout = time.Minute * 120

type cacheItem[T any] struct {
	key     string
	val     *T
	expires time.Time
}

// Cache stores data.
type Cache[T any] struct {
	log   *log.Logger
	name  string
	store *bbolt.DB
}

// New creates and returns a new Cache of the specified type.
func New[T any](name string) (*Cache[T], error) {
	var (
		err    error
		dbname string
		opt    = bbolt.Options{
			Timeout: time.Millisecond * 500,
		}
		c = &Cache[T]{name: name}
	)

	dbname = filepath.Join(common.CachePath, name)

	if c.log, err = common.GetLogger(logdomain.Cache); err != nil {
		return nil, err
	} else if c.store, err = bbolt.Open(dbname, fs.ModePerm, &opt); err != nil {
		c.log.Printf("[CRITICAL] Failed to open BBolt store at %s: %s\n",
			dbname,
			err.Error())
		return nil, err
	}

	return c, nil
} // func New[T any](name string) (*Cache[T], error)

// Store stores a value under the given key.
func (c *Cache[T]) Store(key string, val *T) error {
	var (
		err            error
		item           cacheItem[T]
		valbuf, keybuf []byte
	)

	item = cacheItem[T]{
		key:     key,
		val:     val,
		expires: time.Now().Add(timeout),
	}
	keybuf = []byte(key)
	if valbuf, err = json.Marshal(&item); err != nil {
		c.log.Printf("[ERROR] Failed to serialize Item to JSON: %s\n",
			err.Error())
		return err
	}

	err = c.store.Update(func(tx *bbolt.Tx) error {
		var (
			ex     error
			bucket *bbolt.Bucket
		)

		if bucket, ex = tx.CreateBucketIfNotExists([]byte(c.name)); ex != nil {
			c.log.Printf("[ERROR] Couldn't create or retrieve Bucket %s: %s\n",
				c.name,
				err.Error())
			return ex
		} else if ex = bucket.Put(keybuf, valbuf); ex != nil {
			c.log.Printf("[ERROR] Failed to store Key %s: %s\n",
				key,
				err.Error())
			return ex
		}

		return nil
	})

	return err
} // func (c *Cache[T]) Store(key string, val *T) error

// Load looks up a value under the given key.
func (c *Cache[T]) Load(key string) (*T, error) {
	var (
		err            error
		keybuf, valbuf []byte
		value          *cacheItem[T]
	)

	keybuf = []byte(key)

	err = c.store.View(func(tx *bbolt.Tx) error {
		var bucket *bbolt.Bucket

		if bucket = tx.Bucket([]byte(c.name)); bucket == nil {
			c.log.Printf("[CRITICAL] Bucket %s does not exist in Cache.\n",
				c.name)
		} else if valbuf = bucket.Get(keybuf); valbuf == nil {
			c.log.Printf("[DEBUG] Key %s was not found in Cache.\n",
				c.name)
		}

		return nil
	})

	if err != nil {
		c.log.Printf("[ERROR] Failed to lookup key %q in Cache: %s\n",
			key,
			err.Error())
		return nil, err
	} else if valbuf == nil {
		return nil, nil
	}

	value = new(cacheItem[T])

	if err = json.Unmarshal(valbuf, value); err != nil {
		c.log.Printf("[ERROR] Failed to deserialize value from JSON: %s\n",
			err.Error())
		return nil, err
	} else if value.expires.Before(time.Now()) {
		return nil, nil
	}

	return value.val, nil
} // func (c *Cache[T]) Load(key string) (*T, error)
