// Package engine orchestrates concurrent workers and aggregates load test results.
package engine

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"goperf/internal/config"
	"goperf/internal/worker"
)

// Result holds the aggregated outcome of a load test run.
type Result struct {
	TotalRequests int
	Succeeded     int
	Failed        int
	StatusCodes   map[int]int
	Errors        map[string]int
	Latencies     []time.Duration
	TotalDuration time.Duration
}

// Run executes the load test with the given configuration, launching
// concurrent workers and collecting their results into a single Result.
func Run(cfg config.Config) Result {
	transport := &http.Transport{
		MaxIdleConnsPerHost: cfg.Concurrency,
		DialContext: (&net.Dialer{
			Timeout: cfg.Timeout,
		}).DialContext,
	}
	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	results := make(chan worker.Result, cfg.Concurrency*100)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(cfg.Concurrency)

	for range cfg.Concurrency {
		go func() {
			defer wg.Done()
			worker.Run(ctx, client, cfg.Method, cfg.URL, results)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	start := time.Now()
	res := Result{
		StatusCodes: make(map[int]int),
		Errors:      make(map[string]int),
	}
	for rr := range results {
		res.TotalRequests++
		if rr.Error != nil {
			res.Failed++
			res.Errors[rr.Error.Error()]++
		} else if rr.StatusCode >= 400 {
			res.Failed++
			res.StatusCodes[rr.StatusCode]++
		} else {
			res.Succeeded++
			res.StatusCodes[rr.StatusCode]++
		}
		res.Latencies = append(res.Latencies, rr.Duration)
	}
	res.TotalDuration = time.Since(start)

	return res
}
