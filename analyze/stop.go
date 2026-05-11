// /home/krylon/go/src/github.com/blicero/newsroom/analyze/stop.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-11 11:10:35 krylon>

package analyze

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	root    = "corpus"
	newline = "\n"
)

var stopwords map[string]map[string]bool

func init() {
	var (
		err       error
		stopLists []fs.DirEntry
		namePat   = regexp.MustCompile(`stop_words_(\w{2})[.]txt`)
	)

	stopwords = make(map[string]map[string]bool)

	if stopLists, err = corpus.ReadDir(root); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to read corpora: %s\n",
			err.Error())
	}

	for _, file := range stopLists {
		var (
			lang       string
			contentRaw []byte
			path       = filepath.Join(root, file.Name())
		)

		lang = namePat.FindAllStringSubmatch(file.Name(), -1)[0][1] // nolint: nilaway

		if contentRaw, err = corpus.ReadFile(path); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Cannot read %s: %s\n",
				path,
				err.Error())
		}

		var (
			content = string(contentRaw)
			words   = strings.Split(content, newline)
		)

		stopwords[lang] = make(map[string]bool, len(words))

		for _, w := range words {
			stopwords[lang][w] = true
		}
	}
}
