// /home/krylon/go/src/github.com/blicero/newsroom/analyze/synonyms.go
// -*- mode: go; coding: utf-8; -*-
// Created on 08. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-07-17 16:26:43 krylon>

package analyze

type antithesaurus map[string]string

func (at antithesaurus) substitute(word string) string {
	if sub, ok := at[word]; ok {
		return sub
	}

	return word
} // func (at antithesaurus) substitute(word string) string)

var dict = antithesaurus{
	"US":          "USA",
	"Hormuz":      "Hormus",
	"president":   "President",
	"Deutschland": "Germany",
	"AI":          "Artificial Intelligence",
	"KI":          "Artificial Intelligence",
	"Russland":    "Russia",
	"EU":          "Europa",
	"Frankreich":  "France",
	"Krieg":       "War",
	"Wetter":      "Weather",
	"England":     "UK",
}
