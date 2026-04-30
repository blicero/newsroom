// /home/krylon/go/src/github.com/blicero/newsroom/cache/cache.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-22 12:45:57 krylon>

package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
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
	Key     string
	Val     *T
	Expires time.Time
}

// Cache stores data.
type Cache[T any] struct {
	log   *log.Logger
	name  string
	store *bbolt.DB
}

// New creates and returns a new Cache of the specified type.
func New[T any](name string) (*Cache[T], error) {
	const maxErr = 5

	var (
		err    error
		errCnt int
		dbname string
		opt    = bbolt.Options{
			Timeout: time.Millisecond * 1500,
		}
		c = &Cache[T]{name: name}
	)

	dbname = filepath.Join(common.CachePath, name)

	if c.log, err = common.GetLogger(logdomain.Cache); err != nil {
		return nil, err
	}

OPEN:
	if c.store, err = bbolt.Open(dbname, fs.ModePerm, &opt); err != nil {
		if err.Error() == "timeout" && errCnt < maxErr {
			errCnt++
			time.Sleep(time.Millisecond * 50)
			goto OPEN
		}
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
		citem          cacheItem[T]
		encbuf         bytes.Buffer
		enc            = gob.NewEncoder(&encbuf)
		keybuf, valbuf []byte
	)

	citem = cacheItem[T]{
		Key:     key,
		Val:     val,
		Expires: time.Now().Add(timeout),
	}

	keybuf = []byte(key)

	if err = enc.Encode(&citem); err != nil {
		c.log.Printf("[ERROR] Failed to serialize Item: %s\n",
			err.Error())
		return err
	}

	valbuf = encbuf.Bytes()

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
		decbuf         *bytes.Buffer
		dec            *gob.Decoder
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

	decbuf = bytes.NewBuffer(valbuf)
	dec = gob.NewDecoder(decbuf)

	value = new(cacheItem[T])
	if err = dec.Decode(value); err != nil {
		c.log.Printf("[ERROR] Failed to decode value for key %s: %s\n",
			key,
			err.Error())
	}

	if value.Expires.Before(time.Now()) {
		return nil, nil
	}

	return value.Val, nil
} // func (c *Cache[T]) Load(key string) (*T, error)

// Delete removes the given key from the Cache.
// It is not an error to delete a key that does not exist, in that case no
// change is made to the underlying data store.
func (c *Cache[T]) Delete(key string) error {
	var err error

	if err = c.store.Update(func(tx *bbolt.Tx) error {
		var bucket = tx.Bucket([]byte(c.name))

		return bucket.Delete([]byte(key))
	}); err != nil {
		c.log.Printf("[ERROR] Failed to delete key %s: %s\n",
			key,
			err.Error())
	}

	return err
} // func (c *Cache[T]) Delete(key string) error

// Purge removes expired entries from the Cache. If all is true,
// it removes *all* entries.
func (c *Cache[T]) Purge(all bool) error {
	var (
		err    error
		delCnt int64
		now    = time.Now()
	)

	if err = c.store.Update(func(tx *bbolt.Tx) error {
		var (
			ex     error
			bucket *bbolt.Bucket // = tx.Bucket([]byte(c.name))
			cur    *bbolt.Cursor // = bucket.Cursor()
		)

		if bucket, ex = tx.CreateBucketIfNotExists([]byte(c.name)); ex != nil {
			return fmt.Errorf("failed to create/obtain Bucket %s: %w",
				c.name,
				ex)
		} else if bucket == nil {
			return fmt.Errorf(
				"CreateBucketIfNotExists(%s) did not raise an error but returned a nil bucket",
				c.name)
		}

		cur = bucket.Cursor()

		for key, val := cur.First(); key != nil; key, val = cur.Next() {
			var (
				decbuf *bytes.Buffer
				dec    *gob.Decoder
				citem  *cacheItem[T]
			)

			decbuf = bytes.NewBuffer(val)
			dec = gob.NewDecoder(decbuf)

			citem = new(cacheItem[T])

			if ex = dec.Decode(citem); err != nil {
				c.log.Printf("[ERROR] Cannot decode cached Item %s: %s\n",
					key,
					ex.Error())
				continue
			} else if all || citem.Expires.Before(now) {
				if ex = cur.Delete(); ex != nil {
					c.log.Printf("[ERROR] Cannot delete cached Item %s: %s\n",
						key,
						ex.Error())
					return ex
				}
				delCnt++
			}
		}

		return nil
	}); err != nil {
		c.log.Printf("[ERROR] Failed to purge %s Cache: %s\n",
			c.name,
			err.Error())
		return err
	}

	c.log.Printf("[DEBUG] Successfully purged %d items from %s Cache.\n",
		delCnt,
		c.name)

	return nil
} // func (c *Cache[T]) Purge(all bool) error
