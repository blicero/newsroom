// /home/krylon/go/src/github.com/blicero/newsroom/critic/01_critic_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 30. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-30 20:27:09 krylon>

package critic

import "testing"

var tc *Critic

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
