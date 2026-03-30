// /home/krylon/go/src/github.com/blicero/newsroom/critic/01_critic_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 30. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-30 21:13:45 krylon>

package critic

import (
	"testing"

	"github.com/blicero/newsroom/model"
	"github.com/blicero/newsroom/model/rating"
)

var (
	tfeed *model.Feed
	tc    *Critic
)

func TestCreateCritic(t *testing.T) {
	var (
		err error
	)

	if tc, err = New(); err != nil {
		tc = nil
		t.Fatalf("Failed to create Critic: %s",
			err.Error())
	} else if tc == nil {
		t.Fatal("New() did not return an error, but the Critic is nil!")
	}
} // func TestCreateCritic(t *testing.T)

func TestTrain(t *testing.T) {
	var (
		err error
	)

	if tc == nil {
		t.SkipNow()
	}

	if err = tc.Retrain(); err != nil {
		t.Fatalf("Failed to train Critic: %s",
			err.Error())
	}
} // func TestTrain(t *testing.T)

func TestClassify(t *testing.T) {
	if tc == nil || tfeed == nil {
		t.SkipNow()
	}

	type tstItem struct {
		item           *model.Item
		expectedRating rating.Rating
	}

	var titems = []tstItem{
		tstItem{
			item: &model.Item{
				FeedID: tfeed.ID,
				Title:  "Athlete at Olympic games breaks record",
				Body: `
At the Olympic games in Italy, the Bob team from Spain set a new world record,
their bob sled actually broke the sound barrier.
`,
			},
			expectedRating: rating.Boring,
		},
		tstItem{
			item: &model.Item{
				FeedID: tfeed.ID,
				Title:  "Japan builds new supercomputer",
				Body: `
The Japanese Secretary of Science and Education has announced plans to build a new
supercomputer for simulations of orbital mechanics as well as dark matter and antimatter.
`,
			},
			expectedRating: rating.Interesting,
		},
	}

	for idx, it := range titems {
		var (
			err error
			cls rating.Rating
		)

		if cls, err = tc.Classify(it.item); err != nil {
			t.Errorf("Failed to classify Item %d (%s): %s",
				idx+1,
				it.item.Title,
				err.Error())
		} else if cls != it.expectedRating {
			t.Errorf("Unexpected Rating for Item %d (%s): %s (expected %s)",
				idx+1,
				it.item.Title,
				cls,
				it.expectedRating)
		}
	}
} // func TestClassify(t *testing.T)
