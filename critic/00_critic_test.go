// /home/krylon/go/src/github.com/blicero/newsroom/critic/00_critic_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-30 20:24:11 krylon>

package critic

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/blicero/newsroom/common"
)

func TestMain(m *testing.M) {
	var (
		err     error
		result  int
		baseDir = time.Now().Format("/tmp/newsroom_critic_test_20060102_150405")
	)

	if err = common.SetBaseDir(baseDir); err != nil {
		fmt.Printf("Cannot set base directory to %s: %s\n",
			baseDir,
			err.Error())
		os.Exit(1)
	} else if err = prepare(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to prepare Database: %s\n",
			err.Error())
	} else if result = m.Run(); result == 0 {
		// If any test failed, we keep the test directory (and the
		// database inside it) around, so we can manually inspect it
		// if needed.
		// If all tests pass, OTOH, we can safely remove the directory.
		fmt.Printf("Removing BaseDir %s\n",
			baseDir)
		_ = os.RemoveAll(baseDir)
	} else {
		fmt.Printf(">>> TEST DIRECTORY: %s\n", baseDir)
	}

	os.Exit(result)
} // func TestMain(m *testing.M)
