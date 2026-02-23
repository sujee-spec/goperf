package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goperf/internal/config"
)

func TestRunCompletesAndAggregates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Config{
		URL:         srv.URL,
		Method:      "GET",
		Concurrency: 2,
		Duration:    300 * time.Millisecond,
		Timeout:     5 * time.Second,
	}

	res := Run(cfg)

	if res.TotalRequests == 0 {
		t.Fatal("expected at least one request")
	}
	if res.TotalRequests != res.Succeeded+res.Failed {
		t.Errorf("total %d != succeeded %d + failed %d", res.TotalRequests, res.Succeeded, res.Failed)
	}
	if res.Failed != 0 {
		t.Errorf("expected 0 failures against test server, got %d", res.Failed)
	}
	if len(res.Latencies) != res.TotalRequests {
		t.Errorf("latencies count %d != total requests %d", len(res.Latencies), res.TotalRequests)
	}
	if res.TotalDuration <= 0 {
		t.Error("expected positive total duration")
	}
}

func TestRunCountsFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := config.Config{
		URL:         srv.URL,
		Method:      "GET",
		Concurrency: 1,
		Duration:    200 * time.Millisecond,
		Timeout:     5 * time.Second,
	}

	res := Run(cfg)

	if res.Succeeded != 0 {
		t.Errorf("expected 0 succeeded, got %d", res.Succeeded)
	}
	if res.Failed == 0 {
		t.Error("expected failures for 503 responses")
	}
}
