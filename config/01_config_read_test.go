// /home/krylon/go/src/github.com/blicero/newsroom/config/01_config_read_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 15. 06. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-07-03 11:33:16 krylon>

package config

import (
	"testing"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/davecgh/go-spew/spew"
)

func TestReadNoConfig(t *testing.T) {
	var (
		err    error
		cfg    *Config
		expect = Config{
			Global: Global{
				Debug:        true,
				CacheTimeout: time.Minute * 120,
			},
			Web: Web{
				Address:            "[::]:4200",
				HideBoring:         true,
				HideProbablyBoring: false,
				ItemsPerPage:       100,
				TrendDays:          7,
			},
			Cluster: Cluster{
				Period: time.Hour * 24 * 2,
			},
			Path: Path{
				Base:      "~/.newsroom.d",
				Log:       "newsroom.log",
				Database:  "newsroom.db",
				Cache:     "cache.d",
				Blacklist: "blacklist.db",
			},
			Loglevel: Loglevel{
				Database:   "DEBUG",
				DBPool:     "DEBUG",
				Engine:     "DEBUG",
				Cache:      "DEBUG",
				Critic:     "DEBUG",
				Classifier: "DEBUG",
				Scrub:      "DEBUG",
				Web:        "DEBUG",
				Blacklist:  "DEBUG",
				Analyze:    "DEBUG",
				Main:       "DEBUG",
			},
		}
	)

	if cfg, err = Read(common.CfgPath); err != nil {
		t.Fatalf("Failed to read %s: %s\n",
			common.CfgPath,
			err.Error())
	} else if cfg == nil {
		t.Fatal("Read() did not return a Config object")
	} else if !cfg.Equal(&expect) {
		t.Fatalf("Read() returned unexpected config:\nExpected: %s\nGot: %s\n",
			spew.Sdump(&expect),
			spew.Sdump(cfg))
	}
} // func TestReadNoConfig(t *testing.T)

func TestReadExampleConfig(t *testing.T) {
	const cfgPath = "testdata/test01.toml"
	var (
		err    error
		cfg    *Config
		expect = Config{
			Global: Global{
				Debug:        false,
				CacheTimeout: time.Minute * 360,
			},
			Web: Web{
				Address:            "[::1]:4242",
				HideBoring:         true,
				HideProbablyBoring: true,
				ItemsPerPage:       50,
				TrendDays:          14,
			},
			Cluster: Cluster{
				Period: time.Hour * 72,
			},
			Path: Path{
				Base:      "~/.newsroom.d",
				Log:       "newsroom.log",
				Database:  "newsroom.db",
				Cache:     "cache.d",
				Blacklist: "blacklist.db",
			},
			Loglevel: Loglevel{
				Database:   "DEBUG",
				DBPool:     "DEBUG",
				Engine:     "DEBUG",
				Cache:      "DEBUG",
				Critic:     "DEBUG",
				Classifier: "DEBUG",
				Scrub:      "DEBUG",
				Web:        "DEBUG",
				Blacklist:  "DEBUG",
				Analyze:    "DEBUG",
				Main:       "DEBUG",
			},
		}
	)

	if cfg, err = Read(cfgPath); err != nil {
		t.Fatalf("Failed to read %s: %s\n",
			common.CfgPath,
			err.Error())
	} else if cfg == nil {
		t.Fatal("Read() did not return a Config object")
	} else if !cfg.Equal(&expect) {
		t.Fatalf("Read() returned unexpected config:\nExpected: %s\nGot: %s\n",
			spew.Sdump(&expect),
			spew.Sdump(cfg))
	}
} // func TestReadExampleConfig(t *testing.T)
