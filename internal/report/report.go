// Package report computes latency statistics and prints formatted load test results.
package report

import (
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"goperf/internal/config"
	"goperf/internal/engine"
)

// Stats holds computed latency percentiles and throughput for a load test.
type Stats struct {
	Average time.Duration
	P50     time.Duration
	P90     time.Duration
	P99     time.Duration
	RPS     float64
}

// Compute calculates latency percentiles, average, and requests per second
// from the raw engine result.
func Compute(res engine.Result) Stats {
	if len(res.Latencies) == 0 {
		return Stats{}
	}

	sorted := make([]time.Duration, len(res.Latencies))
	copy(sorted, res.Latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	var total time.Duration
	for _, d := range sorted {
		total += d
	}

	n := len(sorted)
	return Stats{
		Average: total / time.Duration(n),
		P50:     sorted[percentileIndex(n, 50)],
		P90:     sorted[percentileIndex(n, 90)],
		P99:     sorted[percentileIndex(n, 99)],
		RPS:     float64(res.TotalRequests) / res.TotalDuration.Seconds(),
	}
}

// percentileIndex returns the index for the given percentile using the
// nearest-rank method: index = ceil(p/100 * n) - 1.
func percentileIndex(n, p int) int {
	idx := int(math.Ceil(float64(p)/100*float64(n))) - 1
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}

// Print writes a formatted load test report to the given writer.
func Print(w io.Writer, cfg config.Config, res engine.Result) error {
	stats := Compute(res)

	_, err := fmt.Fprintf(w, `
--- goperf results ---
Target:       %s %s
Duration:     %s
Concurrency:  %d

Requests:     %d total, %d succeeded, %d failed

Latency:
  Average:    %s
  P50:        %s
  P90:        %s
  P99:        %s

Throughput:   %.2f req/s
`,
		cfg.Method, cfg.URL,
		res.TotalDuration.Round(time.Millisecond),
		cfg.Concurrency,
		res.TotalRequests, res.Succeeded, res.Failed,
		stats.Average.Round(time.Microsecond),
		stats.P50.Round(time.Microsecond),
		stats.P90.Round(time.Microsecond),
		stats.P99.Round(time.Microsecond),
		stats.RPS,
	)
	if err != nil {
		return err
	}

	if len(res.StatusCodes) > 0 {
		fmt.Fprintf(w, "Status codes:\n")
		for code, count := range res.StatusCodes {
			fmt.Fprintf(w, "  [%d]        %d\n", code, count)
		}
		fmt.Fprintln(w)
	}

	if len(res.Errors) > 0 {
		fmt.Fprintf(w, "Errors:\n")
		for msg, count := range res.Errors {
			fmt.Fprintf(w, "  (%d) %s\n", count, msg)
		}
		fmt.Fprintln(w)
	}

	return nil
}
