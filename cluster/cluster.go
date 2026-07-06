// /home/krylon/go/src/github.com/blicero/newsroom/cluster/cluster.go
// -*- mode: go; coding: utf-8; -*-
// Created on 01. 07. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-07-04 12:07:24 krylon>

// Package cluster uses Latent Semantic Analysis (LSA) to find related Items.
// At least that is the idea.
package cluster

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/Feralthedogg/go-functional/pkg/functional"
	"github.com/blicero/krylib"
	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/database"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/stopwords"
	"github.com/james-bowman/nlp"
	"github.com/james-bowman/nlp/measures/pairwise"
	"gonum.org/v1/gonum/mat"
)

// When looking for related Items, processing ALL Items in the
// Database is not going to be practical.
//
// For starters, I will look into Items that were added within, say, a
// week of the one we are starting with.

var Period = time.Hour * 24 * 7

// Match represents a related Document.
type Match struct {
	Item       *model.Item
	Similarity float64
}

// SemanticCluster represents an Item and a group of Items that are similar
// or related.
type SemanticCluster struct {
	Root    *model.Item
	Related []Match
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
		feed       *model.Feed
		pipe       *nlp.Pipeline
		stop       []string
		stripped   []string
	)

	begin = item.Timestamp.Add(-Period)
	end = item.Timestamp.Add(Period)

	if items, err = s.db.ItemGetByPeriod(begin, end); err != nil {
		s.log.Printf("[ERROR] Failed to find Items by Period [%s - %s]: %s\n",
			begin.Format(common.TimestampFormatMinute),
			end.Format(common.TimestampFormatMinute),
			err.Error())
		return nil, err
	} else if feed, err = s.db.FeedGetByID(item.FeedID); err != nil {
		s.log.Printf("[ERROR] Failed to lookup Feed for Item %q (%d): %s\n",
			item.Title,
			item.ID,
			err.Error())
		return nil, err
	} else if feed == nil {
		s.log.Printf("[CANTHAPPEN] No Feed was found for Item %d (%q)\n",
			item.ID,
			item.Title)
		return nil, krylib.ErrInvalidValue
	}

	stop = stopwords.GetWords(feed.Language)

	stripped = functional.Map(
		func(i *model.Item) string { return i.Strip() },
		items)

	pipe = nlp.NewPipeline(
		nlp.NewCountVectoriser(stop...),
		nlp.NewTfidfTransformer(),
		nlp.NewLatentDirichletAllocation(16),
	)

	// First we train the model
	var lsi mat.Matrix

	if lsi, err = pipe.FitTransform(stripped...); err != nil {
		s.log.Printf("[ERROR] Failed to process Items for clustering: %s\n",
			err.Error())
		return nil, err
	}

	var queryMat mat.Matrix

	queryMat, _ = pipe.Transform(item.Strip())

	var matches = s.calcCosine(queryMat, lsi, items, item.Title)

	slices.SortFunc(matches, func(m1, m2 Match) int {
		if m1.Similarity > m2.Similarity {
			return -1
		} else if m1.Similarity < m2.Similarity {
			return 1
		}

		return 0
	})

	return &SemanticCluster{
		Root:    item,
		Related: matches,
	}, nil
} // func (s *Scout) FindCluster(item *model.Item) (*SemanticCluster, error)

func (s *Scout) calcCosine(query mat.Matrix, tdmat mat.Matrix, corpus []*model.Item, name string) []Match {
	// iterate over document feature vectors (columns) in the LSI and
	// compare with the query vector for similarity.  Similarity is determined
	// by the difference between the angles of the vectors known as the cosine
	// similarity
	_, docs := tdmat.Dims()

	var matches = make([]Match, 0)

	fmt.Printf("Comparing based on %s\n", name)

	for i := 0; i < docs; i++ {
		queryVec := query.(mat.ColViewer).ColView(0)
		docVec := tdmat.(mat.ColViewer).ColView(i)
		similarity := pairwise.CosineSimilarity(queryVec, docVec)

		if similarity < 0.5 {
			continue
		}

		s.log.Printf("[DEBUG] Comparing '%s' = %.1f\n", corpus[i].Title, similarity)
		var match = Match{
			Item:       corpus[i],
			Similarity: similarity,
		}

		matches = append(matches, match)
	}

	return matches
} // func (s *Scout) calcCosine(query mat.Matrix, tdmat mat.Matrix, corpus []*model.Item, name string) []Match
