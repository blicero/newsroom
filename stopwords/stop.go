// /home/krylon/go/src/github.com/blicero/newsroom/stopwords/stop.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-07-01 11:57:57 krylon>

package stopwords

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/exp/slices"
)

//go:embed corpus
var corpus embed.FS

const (
	root    = "corpus"
	newline = "\n"
)

// StopWords is a map of language -> (stopword -> bool)
var StopWords map[string]map[string]bool

func init() {
	var (
		err       error
		stopLists []fs.DirEntry
		namePat   = regexp.MustCompile(`stop_words_(\w{2})[.]txt`)
	)

	StopWords = make(map[string]map[string]bool)

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

		StopWords[lang] = make(map[string]bool, len(words))

		for _, w := range words {
			StopWords[lang][w] = true
		}
	}
}

// GetWords returns a slice of stop words for the given language, or nil
// if no words are found for that language.
func GetWords(lng string) []string {
	var (
		wordmap map[string]bool
		ok      bool
	)

	if wordmap, ok = StopWords[lng]; !ok {
		return nil
	}

	var words = make([]string, 0, len(wordmap))

	for word := range wordmap {
		words = append(words, word)
	}

	// FIXME Emacs' Flymake tells me I should inline this call. I don't
	//       know how, and it doesn't give me any assistance, so we'll
	//       ignore this for now.
	slices.Sort(words)
	return words
} // func GetWords(lng string) []string
