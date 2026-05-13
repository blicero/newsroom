// /home/krylon/go/src/github.com/blicero/newsroom/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-13 13:39:30 krylon>

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

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/engine"
	"github.com/blicero/newsroom/web"
)

func main() {
	var (
		err     error
		eng     *engine.Engine
		srv     *web.Server
		ticker  *time.Ticker
		addr    string
		profOut string
		sigQ    = make(chan os.Signal, 1)
	)

	fmt.Printf("%s %s, built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))

	flag.StringVar(&addr, "addr", fmt.Sprintf("[::1]:%d", common.WebPort), "The IP address for the web UI to listen on")
	flag.StringVar(&profOut, "prof", "", "if non-empty, write profiling information to the named file")
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
