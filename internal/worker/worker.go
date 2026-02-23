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
			results <- Result{
				Duration: time.Since(start),
				Error:    err,
			}
			continue
		}

		resp, err := client.Do(req)
		elapsed := time.Since(start)

		if err != nil {
			if ctx.Err() != nil {
				return
			}
			results <- Result{
				Duration: elapsed,
				Error:    err,
			}
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		results <- Result{
			Duration:   elapsed,
			StatusCode: resp.StatusCode,
		}
	}
}
