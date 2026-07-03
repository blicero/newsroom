// /home/krylon/go/src/github.com/blicero/newsroom/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-07-03 11:41:25 krylon>

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/blicero/newsroom/cache"
	"github.com/blicero/newsroom/cluster"
	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/config"
	"github.com/blicero/newsroom/engine"
	"github.com/blicero/newsroom/logdomain"
	"github.com/blicero/newsroom/web"
	"github.com/hashicorp/logutils"
)

func main() {
	var (
		err              error
		eng              *engine.Engine
		srv              *web.Server
		ticker           *time.Ticker
		addr             string
		profOut, baseDir string
		cfg              *config.Config
		sigQ             = make(chan os.Signal, 1)
	)

	fmt.Printf("%s %s, built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))

	baseDir = common.BaseDir

	if baseDir != common.BaseDir {
		if err = common.SetBaseDir(baseDir); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Failed to set BaseDir to %q: %s\n",
				baseDir,
				err.Error(),
			)
			os.Exit(1)
		}
	} else if cfg, err = config.Read(common.CfgPath); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to read config from %s: %s\n",
			common.CfgPath,
			err.Error())
		os.Exit(1)
	}

	addr = cfg.Web.Address

	common.PackageLevels[logdomain.Database] = logutils.LogLevel(cfg.Loglevel.Database)
	common.PackageLevels[logdomain.DBPool] = logutils.LogLevel(cfg.Loglevel.DBPool)
	common.PackageLevels[logdomain.Engine] = logutils.LogLevel(cfg.Loglevel.Engine)
	common.PackageLevels[logdomain.Cache] = logutils.LogLevel(cfg.Loglevel.Cache)
	common.PackageLevels[logdomain.Critic] = logutils.LogLevel(cfg.Loglevel.Critic)
	common.PackageLevels[logdomain.Classifier] = logutils.LogLevel(cfg.Loglevel.Classifier)
	common.PackageLevels[logdomain.Scrub] = logutils.LogLevel(cfg.Loglevel.Scrub)
	common.PackageLevels[logdomain.Web] = logutils.LogLevel(cfg.Loglevel.Web)
	common.PackageLevels[logdomain.Blacklist] = logutils.LogLevel(cfg.Loglevel.Blacklist)
	common.PackageLevels[logdomain.Analyze] = logutils.LogLevel(cfg.Loglevel.Analyze)
	common.PackageLevels[logdomain.Main] = logutils.LogLevel(cfg.Loglevel.Main)

	common.Debug = cfg.Global.Debug
	cache.Timeout = cfg.Global.CacheTimeout
	cluster.Period = cfg.Cluster.Period

	flag.StringVar(&addr, "addr", fmt.Sprintf("[::1]:%d", common.WebPort), "The IP address for the web UI to listen on")
	flag.StringVar(&profOut, "prof", "", "if non-empty, write profiling information to the named file")
	flag.StringVar(&baseDir, "base", common.BaseDir, "directory where application-specific files live")
	flag.Parse()

	if profOut != "" {
		var profH *os.File

		fmt.Printf("Writing profiling data to %s\n",
			profOut)

		if profH, err = os.Create(profOut); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Failed to open %s: %s\n",
				profOut,
				err.Error())
			os.Exit(1)
		}

		defer profH.Close() // nolint: errcheck

		if err = pprof.StartCPUProfile(profH); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Error starting CPU profile: %s\n",
				err.Error())
			os.Exit(1)
		}

		defer pprof.StopCPUProfile()
	}

	if eng, err = engine.Create(runtime.NumCPU()); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create Engine: %s\n",
			err.Error())
		os.Exit(1)
	} else if srv, err = web.Create(addr, eng); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create web server: %s\n",
			err.Error())
		os.Exit(1)
	}

	eng.Start()
	go srv.Run()

	ticker = time.NewTicker(common.ActiveTimeout)
	defer ticker.Stop()

	signal.Notify(sigQ, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			if !eng.IsActive() {
				return
			}
		case s := <-sigQ:
			fmt.Fprintf(
				os.Stderr,
				"Caught signal: %s\n",
				s)
			eng.Stop()
			return
		}
	}
}
