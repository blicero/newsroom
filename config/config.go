// /home/krylon/go/src/github.com/blicero/newsroom/config/config.go
// -*- mode: go; coding: utf-8; -*-
// Created on 29. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-06-30 11:28:26 krylon>

// Package config defines settings that can be modified by a user.
package config

import (
	"io"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/blicero/krylib"
)

const defaultConf = `# Time-stamp: <>

[Global]
Debug = true
CacheTimeout = "2h"

[Web]
Address = "[::]:4200"
HideBoring = true
HideProbablyBoring = false
ItemsPerPage = 100
TrendDays = 7

[Path]
Base = "~/.newsroom.d"
Log = "newsroom.log"
Database = "newsroom.db"
Cache = "cache.d"
Blacklist = "blacklist.db"

[Loglevel]
Database = "DEBUG"
DBPool = "DEBUG"
Engine = "DEBUG"
Cache = "DEBUG"
Critic = "DEBUG"
Classifier = "DEBUG"
Scrub = "DEBUG"
Web = "DEBUG"
Blacklist = "DEBUG"
Analyze = "DEBUG"
Main = "DEBUG"
`

// Web groups the settings for the Web interface.
type Web struct {
	Address            string
	HideBoring         bool
	HideProbablyBoring bool
	ItemsPerPage       int64
	TrendDays          int64
}

// Equal returns true if the receiver is equal to the argument.
func (w *Web) Equal(other any) bool {
	var (
		w2 *Web
		ok bool
	)

	if w2, ok = other.(*Web); !ok {
		return false
	}

	return w.Address == w2.Address &&
		w.HideBoring == w2.HideBoring &&
		w.HideProbablyBoring == w2.HideProbablyBoring &&
		w.ItemsPerPage == w2.ItemsPerPage &&
		w.TrendDays == w2.TrendDays
}

// Path contains the paths to various files and folders.
type Path struct {
	Base      string
	Log       string
	Database  string
	Cache     string
	Blacklist string
}

// Equal returns true if the receiver is equal to the argument.
func (p *Path) Equal(other any) bool {
	var (
		p2 *Path
		ok bool
	)

	if p2, ok = other.(*Path); !ok {
		return false
	}

	return p.Base == p2.Base &&
		p.Log == p2.Log &&
		p.Database == p2.Database &&
		p.Cache == p2.Cache &&
		p.Blacklist == p2.Blacklist
}

// Loglevel contains the log levels for the various subsystems.
type Loglevel struct {
	Database   string
	DBPool     string
	Engine     string
	Cache      string
	Critic     string
	Classifier string
	Scrub      string
	Web        string
	Blacklist  string
	Analyze    string
	Main       string
}

// Equal returns true if the receiver is equal to the argument.
func (l *Loglevel) Equal(other any) bool {
	var (
		l2 *Loglevel
		ok bool
	)

	if l2, ok = other.(*Loglevel); !ok {
		return false
	}

	return l.Database == l2.Database &&
		l.DBPool == l2.DBPool &&
		l.Engine == l2.Engine &&
		l.Cache == l2.Cache &&
		l.Critic == l2.Critic &&
		l.Classifier == l2.Classifier &&
		l.Scrub == l2.Scrub &&
		l.Web == l2.Web &&
		l.Blacklist == l2.Blacklist &&
		l.Analyze == l2.Analyze &&
		l.Main == l2.Main
}

// Global contains settings that don't fit anywhere else.
type Global struct {
	Debug        bool
	CacheTimeout time.Duration
}

// Equal returns true if the receiver is equal to the argument.
func (g *Global) Equal(other any) bool {
	var (
		g2 *Global
		ok bool
	)

	if g2, ok = other.(*Global); !ok {
		return false
	}

	return g.Debug == g2.Debug &&
		g.CacheTimeout == g2.CacheTimeout
} // func (g *Global) Equal(other any) bool

// Config defines the variables that a user can set.
type Config struct {
	Global   Global
	Web      Web
	Path     Path
	Loglevel Loglevel
}

// Equal returns true if the receiver is equal to the argument.
func (c *Config) Equal(other any) bool {
	var (
		c2 *Config
		ok bool
	)

	if c2, ok = other.(*Config); !ok {
		return false
	}

	return c.Global.Equal(&c2.Global) &&
		c.Web.Equal(&c2.Web) &&
		c.Path.Equal(&c2.Path) &&
		c.Loglevel.Equal(&c2.Loglevel)
} // func (c *Config) Equal(other any) bool

// Read attempts to read the configuration from the given file.
func Read(path string) (*Config, error) {
	var (
		err    error
		exists bool
		fh     *os.File
		buf    []byte
		cfg    = new(Config)
	)

	if exists, err = krylib.Fexists(path); err != nil {
		return nil, err
	} else if !exists {
		if err = writeDefaultCfg(path); err != nil {
			return nil, err
		}
	}

	if fh, err = os.Open(path); err != nil {
		return nil, err
	}

	defer fh.Close()

	if buf, err = io.ReadAll(fh); err != nil {
		return nil, err
	} else if err = toml.Unmarshal(buf, cfg); err != nil {
		return nil, err
	}

	if cfg.Loglevel.Database == "" {
		cfg.Loglevel.Database = "DEBUG"
	}

	if cfg.Loglevel.DBPool == "" {
		cfg.Loglevel.DBPool = "DEBUG"
	}

	if cfg.Loglevel.Engine == "" {
		cfg.Loglevel.Engine = "DEBUG"
	}

	if cfg.Loglevel.Cache == "" {
		cfg.Loglevel.Cache = "DEBUG"
	}

	if cfg.Loglevel.Critic == "" {
		cfg.Loglevel.Critic = "DEBUG"
	}

	if cfg.Loglevel.Classifier == "" {
		cfg.Loglevel.Classifier = "DEBUG"
	}

	if cfg.Loglevel.Scrub == "" {
		cfg.Loglevel.Scrub = "DEBUG"
	}

	if cfg.Loglevel.Web == "" {
		cfg.Loglevel.Web = "DEBUG"
	}

	if cfg.Loglevel.Blacklist == "" {
		cfg.Loglevel.Blacklist = "DEBUG"
	}

	if cfg.Loglevel.Analyze == "" {
		cfg.Loglevel.Analyze = "DEBUG"
	}

	if cfg.Loglevel.Main == "" {
		cfg.Loglevel.Main = "DEBUG"
	}

	return cfg, nil
} // func Read(path string) (*Config, error)

func writeDefaultCfg(path string) error {
	var (
		err error
		fh  *os.File
	)

	if fh, err = os.Create(path); err != nil {
		return err
	}

	defer fh.Close()

	if _, err = fh.Write([]byte(defaultConf)); err != nil {
		return err
	}

	return nil
} // func writeDefaultCfg(path string) error
