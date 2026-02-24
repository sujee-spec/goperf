// Package worker implements the HTTP request loop for a single load-test worker.
package worker

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Result holds the outcome of a single HTTP request.
type Result struct {
	Duration   time.Duration
	StatusCode int
	Error      error
}

// Run sends HTTP requests in a loop until the context is cancelled.
// Each result is sent to the results channel.
func Run(ctx context.Context, client *http.Client, method, url string, results chan<- Result) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			select {
			case results <- Result{
				Duration: time.Since(start),
				Error:    err,
			}:
			case <-ctx.Done():
				return
			}
			continue
		}

		resp, err := client.Do(req)
		elapsed := time.Since(start)

		if err != nil {
			// Record the error even if context was cancelled mid-request.
			// This ensures in-flight requests at shutdown are counted.
			select {
			case results <- Result{
				Duration: elapsed,
				Error:    err,
			}:
			case <-ctx.Done():
				// Channel full and context done — best-effort, drop it.
				return
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}

		func() {
			defer resp.Body.Close()
			io.Copy(io.Discard, resp.Body)
		}()

		select {
		case results <- Result{
			Duration:   elapsed,
			StatusCode: resp.StatusCode,
		}:
		case <-ctx.Done():
			return
		}
	}
}
