// /home/krylon/go/src/github.com/blicero/newsroom/scrub/scrub.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 04. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-01 16:14:01 krylon>

// Package scrub cleans up HTML
package scrub

import (
	"bytes"
	"log"

	"github.com/PuerkitoBio/goquery"
	"github.com/blicero/newsroom/cache"
	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"golang.org/x/net/html"
)

// Scrubber sanitizes the HTML from RSS Items.
type Scrubber struct {
	log   *log.Logger
	cache *cache.Cache[string]
}

// Create instantiates a new Scrubber and returns it.
func Create() (*Scrubber, error) {
	var (
		err error
		s   = new(Scrubber)
	)

	if s.log, err = common.GetLogger(logdomain.Scrub); err != nil {
		return nil, err
	} else if s.cache, err = cache.New[string]("scrub"); err != nil {
		s.log.Printf("[ERROR] Failed to open cache: %s\n",
			err.Error())
		return nil, err
	}

	return s, nil
} // func Create() (*Scrubber, error)

// Scrub cleans up an Item's HTML.
func (s *Scrubber) Scrub(item *model.Item) error {
	var (
		err  error
		doc  *goquery.Document
		sel  *goquery.Selection
		node *html.Node
		buf  = bytes.NewBufferString(item.Body)
	)

	if node, err = html.Parse(buf); err != nil {
		s.log.Printf("[ERROR] Failed to parse HTML from body of Item %d (%s): %s\n\nRaw Body: %s\n\n",
			item.ID,
			item.Title,
			err.Error(),
			item.Body)
		return err
	}

	doc = goquery.NewDocumentFromNode(node)
	sel = doc.Find("a")

	sel.Each(func(idx int, fragment *goquery.Selection) {
		elt := fragment.Nodes[0]
		elt.Attr = append(elt.Attr,
			html.Attribute{
				Key: "target",
				Val: "_blank",
			})
	})

	doc.Find("script").Remove()
	buf.Reset()

	if err = goquery.Render(buf, doc.Selection); err != nil {
		s.log.Printf("[ERROR] Failed to render goquery doc back to HTML: %s\n",
			err.Error())
		return err
	}

	item.Body = buf.String()
	return nil
} // func (s *Scrubber) Scrub(item *model.Item) error
