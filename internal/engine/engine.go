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

type Result struct {
	TotalRequests int
	Succeeded     int
	Failed        int
	Latencies     []time.Duration
	TotalDuration time.Duration
}

func Run(cfg config.Config) (Result, error) {
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
	var res Result
	for rr := range results {
		res.TotalRequests++
		if rr.Error != nil || rr.StatusCode >= 400 {
			res.Failed++
		} else {
			res.Succeeded++
		}
		res.Latencies = append(res.Latencies, rr.Duration)
	}
	res.TotalDuration = time.Since(start)

	return res, nil
}
