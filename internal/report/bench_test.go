package report

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"goperf/internal/config"
	"goperf/internal/engine"
)

func makeResult(n int) engine.Result {
	latencies := make([]time.Duration, n)
	for i := range latencies {
		latencies[i] = time.Duration(rand.Intn(500)+1) * time.Millisecond
	}
	return engine.Result{
		TotalRequests: n,
		Succeeded:     n,
		StatusCodes:   map[int]int{200: n},
		Errors:        map[string]int{},
		Latencies:     latencies,
		TotalDuration: 10 * time.Second,
	}
}

func BenchmarkCompute100(b *testing.B) {
	res := makeResult(100)
	b.ResetTimer()
	for b.Loop() {
		Compute(res)
	}
}

func BenchmarkCompute1000(b *testing.B) {
	res := makeResult(1_000)
	b.ResetTimer()
	for b.Loop() {
		Compute(res)
	}
}

func BenchmarkCompute10000(b *testing.B) {
	res := makeResult(10_000)
	b.ResetTimer()
	for b.Loop() {
		Compute(res)
	}
}

func BenchmarkCompute100000(b *testing.B) {
	res := makeResult(100_000)
	b.ResetTimer()
	for b.Loop() {
		Compute(res)
	}
}

func BenchmarkPrint(b *testing.B) {
	cfg := config.Config{
		URL:         "http://example.com",
		Method:      "GET",
		Concurrency: 10,
		Duration:    10 * time.Second,
		Timeout:     10 * time.Second,
	}
	res := makeResult(1_000)
	var buf bytes.Buffer

	b.ResetTimer()
	for b.Loop() {
		buf.Reset()
		if err := Print(&buf, cfg, res); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
