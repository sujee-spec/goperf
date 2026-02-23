package report

import (
	"fmt"
	"io"
	"sort"
	"time"

	"goperf/internal/config"
	"goperf/internal/engine"
)

type Stats struct {
	Average time.Duration
	P50     time.Duration
	P90     time.Duration
	P99     time.Duration
	RPS     float64
}

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

func percentileIndex(n, p int) int {
	idx := (n*p)/100 - 1
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}

func Print(w io.Writer, cfg config.Config, res engine.Result) {
	stats := Compute(res)

	fmt.Fprintf(w, "\n--- goperf results ---\n")
	fmt.Fprintf(w, "Target:       %s %s\n", cfg.Method, cfg.URL)
	fmt.Fprintf(w, "Duration:     %s\n", res.TotalDuration.Round(time.Millisecond))
	fmt.Fprintf(w, "Concurrency:  %d\n", cfg.Concurrency)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Requests:     %d total, %d succeeded, %d failed\n", res.TotalRequests, res.Succeeded, res.Failed)
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Latency:\n")
	fmt.Fprintf(w, "  Average:    %s\n", stats.Average.Round(time.Microsecond))
	fmt.Fprintf(w, "  P50:        %s\n", stats.P50.Round(time.Microsecond))
	fmt.Fprintf(w, "  P90:        %s\n", stats.P90.Round(time.Microsecond))
	fmt.Fprintf(w, "  P99:        %s\n", stats.P99.Round(time.Microsecond))
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Throughput:   %.2f req/s\n", stats.RPS)
}
