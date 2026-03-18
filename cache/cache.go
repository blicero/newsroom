// /home/krylon/go/src/github.com/blicero/newsroom/cache/cache.go
// -*- mode: go; coding: utf-8; -*-
// Created on 18. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-18 14:30:25 krylon>

package cache

import (
	"bytes"
	"encoding/gob"
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

	// if valbuf, err = json.Marshal(&citem); err != nil {
	// 	c.log.Printf("[ERROR] Failed to serialize Item to JSON: %s\n",
	// 		err.Error())
	// 	return err
	// }
	if err = enc.Encode(&citem); err != nil {
		c.log.Printf("[ERROR] Failed to serialize Item: %s\n",
			err.Error())
		return err
	}

	valbuf = encbuf.Bytes()

	// if common.Debug {
	// 	c.log.Printf("[DEBUG] Raw data:\nKey: %s\nValue: %#v\n\n",
	// 		key,
	// 		val)
	// 	c.log.Printf("[DEBUG] Serialized data:\nKey: %s\nValue: %#v\n\n",
	// 		keybuf,
	// 		valbuf)
	// }

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
	} /*else if common.Debug {
		c.log.Printf("[DEBUG] Retrieved JSON data for key %s:\n%s\n\n",
			key,
			valbuf)
	}*/

	decbuf = bytes.NewBuffer(valbuf)
	dec = gob.NewDecoder(decbuf)

	value = new(cacheItem[T])
	if err = dec.Decode(value); err != nil {
		c.log.Printf("[ERROR] Failed to decode value for key %s: %s\n",
			key,
			err.Error())
	}

	// if err = json.Unmarshal(valbuf, value); err != nil {
	// 	c.log.Printf("[ERROR] Failed to deserialize value from JSON: %s\n",
	// 		err.Error())
	// 	return nil, err
	// } else

	if value.Expires.Before(time.Now()) {
		return nil, nil
	}

	return value.Val, nil
} // func (c *Cache[T]) Load(key string) (*T, error)
