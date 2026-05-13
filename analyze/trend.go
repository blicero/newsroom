// /home/krylon/go/src/github.com/blicero/newsroom/analyze/trend.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 05. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-05-13 12:33:18 krylon>

package analyze

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blicero/newsroom/common"
)

// Series is a list of word frequencies over a longer time span.
type Series struct {
	Periods     []Period
	Frequencies map[string][]float64
}

// ComputeSeries computes a series of frequencies.
func (ts *TrendScout) ComputeSeries(interval time.Duration, icnt, wcnt int) (*Series, error) {
	var (
		err        error
		errCnt     atomic.Int32
		wg         sync.WaitGroup
		begin, end time.Time
		freqs      []WordList
		series     = &Series{
			Periods: make([]Period, icnt),
		}
	)

	end = time.Now().Truncate(time.Hour * 24)
	begin = end.Add(-interval)

	end = end.Add(time.Second * 86400)

	ts.log.Printf("[TRACE] Compute Series %s -- %s\n",
		begin.Format(common.TimestampFormat),
		end.Format(common.TimestampFormat))

	for i := icnt - 1; i >= 0; i-- {
		series.Periods[i] = Period{Begin: begin, End: end}
		end = begin
		begin = end.Add(-interval)
	}

	freqs = make([]WordList, icnt)

	for i, p := range series.Periods {
		wg.Go(func() {
			var (
				ex   error
				hist WordList
			)

			if hist, ex = ts.AnalyzePeriod(&p, wcnt); ex != nil {
				errCnt.Add(1)
				ts.log.Printf("[ERROR] Failed to analyze period %s to %s: %s\n",
					p.Begin.Format(common.TimestampFormatDate),
					p.End.Format(common.TimestampFormatDate),
					ex.Error())
			} else {
				freqs[i] = hist
			}
		})
	}

	wg.Wait()

	if numErr := errCnt.Load(); numErr > 0 {
		err = fmt.Errorf("%d errors occured during analysis of periods", numErr)
		ts.log.Printf("[ERROR] %s\n",
			err.Error())
		return nil, err
	}

	series.Frequencies = make(map[string][]float64)

	for idx, hist := range freqs {
		for _, word := range hist {
			var ok bool

			if _, ok = series.Frequencies[word.Word]; !ok {
				series.Frequencies[word.Word] =
					make([]float64, icnt)
			}

			series.Frequencies[word.Word][idx] = word.Count
		}
	}

	return series, nil
} // func (ts *TrendScout) ComputeSeries(interval time.Duration, icnt, wcnt int) (*Series, error)
