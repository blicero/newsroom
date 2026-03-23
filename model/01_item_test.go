// /home/krylon/go/src/github.com/blicero/newsroom/model/01_item_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 23. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-23 18:01:46 krylon>

package model

import "testing"

func TestStrip(t *testing.T) {
	type testCase struct {
		item     Item
		expected string
	}

	var cases = []testCase{
		{
			Item{
				Title: "Test 01",
				Body:  "<h1>Test 01</h1> This is a test",
			},
			"Test 01 Test 01 This is a test",
		},
	}

	for idx, c := range cases {
		var result string

		result = c.item.Strip()

		if result != c.expected {
			t.Errorf(`Strip of Item %d returned unexpected result:
Expected: %s
Actual:   %s
`,
				idx+1,
				c.expected,
				result)
		}
	}
} // func TestStrip(t *testing.T)
