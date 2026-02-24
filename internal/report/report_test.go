package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"goperf/internal/config"
	"goperf/internal/engine"
)

func TestComputePercentiles(t *testing.T) {
	// Create 100 latencies: 1ms, 2ms, ..., 100ms
	latencies := make([]time.Duration, 100)
	for i := range latencies {
		latencies[i] = time.Duration(i+1) * time.Millisecond
	}

	res := engine.Result{
		TotalRequests: 100,
		Succeeded:     100,
		Latencies:     latencies,
		TotalDuration: 1 * time.Second,
	}

	stats := Compute(res)

	// nearest-rank: ceil(p/100 * n) - 1
	// P50: ceil(0.50*100)-1 = 49 → 50ms
	// P90: ceil(0.90*100)-1 = 89 → 90ms
	// P99: ceil(0.99*100)-1 = 98 → 99ms
	if stats.P50 != 50*time.Millisecond {
		t.Errorf("P50 = %v, want 50ms", stats.P50)
	}
	if stats.P90 != 90*time.Millisecond {
		t.Errorf("P90 = %v, want 90ms", stats.P90)
	}
	if stats.P99 != 99*time.Millisecond {
		t.Errorf("P99 = %v, want 99ms", stats.P99)
	}
	if stats.P50 > stats.P90 || stats.P90 > stats.P99 {
		t.Errorf("percentiles should be ordered: P50=%v P90=%v P99=%v", stats.P50, stats.P90, stats.P99)
	}

	expectedAvg := 50500 * time.Microsecond // (1+100)/2 = 50.5ms
	if stats.Average != expectedAvg {
		t.Errorf("Average = %v, want %v", stats.Average, expectedAvg)
	}

	if stats.RPS != 100.0 {
		t.Errorf("RPS = %v, want 100.0", stats.RPS)
	}
}

func TestComputeEmptyLatencies(t *testing.T) {
	res := engine.Result{}
	stats := Compute(res)

	if stats.Average != 0 || stats.P50 != 0 || stats.RPS != 0 {
		t.Errorf("expected zero stats for empty latencies, got %+v", stats)
	}
}

func TestComputeSingleRequest(t *testing.T) {
	res := engine.Result{
		TotalRequests: 1,
		Succeeded:     1,
		Latencies:     []time.Duration{5 * time.Millisecond},
		TotalDuration: 1 * time.Second,
	}

	stats := Compute(res)

	if stats.P50 != 5*time.Millisecond {
		t.Errorf("P50 = %v, want 5ms", stats.P50)
	}
	if stats.P90 != 5*time.Millisecond {
		t.Errorf("P90 = %v, want 5ms", stats.P90)
	}
}

func TestPrintContainsExpectedSections(t *testing.T) {
	cfg := config.Config{
		URL:         "http://example.com",
		Method:      "GET",
		Concurrency: 5,
		Duration:    10 * time.Second,
		Timeout:     10 * time.Second,
	}

	res := engine.Result{
		TotalRequests: 100,
		Succeeded:     95,
		Failed:        5,
		StatusCodes:   map[int]int{200: 95, 500: 5},
		Errors:        map[string]int{},
		Latencies:     make([]time.Duration, 100),
		TotalDuration: 10 * time.Second,
	}
	for i := range res.Latencies {
		res.Latencies[i] = time.Duration(i+1) * time.Millisecond
	}

	var buf bytes.Buffer
	err := Print(&buf, cfg, res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()

	expected := []string{
		"goperf results",
		"GET http://example.com",
		"Concurrency:  5",
		"100 total",
		"95 succeeded",
		"5 failed",
		"P50:",
		"P90:",
		"P99:",
		"Throughput:",
		"req/s",
	}

	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("output missing %q\nfull output:\n%s", s, output)
		}
	}
}
