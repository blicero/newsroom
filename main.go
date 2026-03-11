// /home/krylon/go/src/github.com/blicero/newsroom/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-11 16:07:46 krylon>

package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/blicero/newsroom/common"
	"github.com/blicero/newsroom/engine"
)

func main() {
	var (
		err    error
		eng    *engine.Engine
		ticker *time.Ticker
		sigQ   = make(chan os.Signal, 1)
	)

	fmt.Printf("%s %s, built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimestampFormat))

	fmt.Println("Nothing to see here (yet), move along!")

	if eng, err = engine.Create(runtime.NumCPU()); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to create Engine: %s\n",
			err.Error())
		os.Exit(1)
	}

	eng.Start()

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
