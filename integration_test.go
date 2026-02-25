package goperf_test

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"goperf/internal/config"
	"goperf/internal/engine"
	"goperf/internal/report"
)

func runFullPipeline(t *testing.T, args []string) (engine.Result, string) {
	t.Helper()

	cfg, err := config.Parse(args)
	if err != nil {
		t.Fatalf("config parse: %v", err)
	}

	res := engine.Run(cfg)

	var buf bytes.Buffer
	if err := report.Print(&buf, cfg, res); err != nil {
		t.Fatalf("report print: %v", err)
	}

	return res, buf.String()
}

func containsError(errors map[string]int, substrs ...string) bool {
	for msg := range errors {
		lower := strings.ToLower(msg)
		for _, s := range substrs {
			if strings.Contains(lower, s) {
				return true
			}
		}
	}
	return false
}

func TestIntegration_FullSuccessfulRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res, output := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "5",
		"-duration", "200ms",
	})

	if res.TotalRequests == 0 {
		t.Fatal("expected at least one request")
	}
	if res.Succeeded == 0 {
		t.Fatal("expected some successful requests")
	}

	maxAllowed := res.TotalRequests / 100
	if maxAllowed < 2 {
		maxAllowed = 2
	}
	if res.Failed > maxAllowed {
		t.Errorf("too many failures: got %d out of %d total (max allowed %d)",
			res.Failed, res.TotalRequests, maxAllowed)
	}

	for _, section := range []string{
		"goperf results",
		"Requests:",
		"Latency:",
		"P50:", "P90:", "P99:",
		"Throughput:",
		"Status codes:",
		"[200]",
	} {
		if !strings.Contains(output, section) {
			t.Errorf("report output missing %q", section)
		}
	}
}

func TestIntegration_MixedStatusCodes(t *testing.T) {
	var counter atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := counter.Add(1)
		if n%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	res, output := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "2",
		"-duration", "200ms",
	})

	if res.Succeeded == 0 {
		t.Error("expected some successful requests")
	}
	if res.Failed == 0 {
		t.Error("expected some failed requests")
	}
	if got, want := res.Succeeded+res.Failed, res.TotalRequests; got != want {
		t.Errorf("succeeded (%d) + failed (%d) = %d, want total %d",
			res.Succeeded, res.Failed, got, want)
	}

	wantCodes := []int{200, 500}
	for _, code := range wantCodes {
		if _, ok := res.StatusCodes[code]; !ok {
			t.Errorf("result missing status code %d", code)
		}
		if !strings.Contains(output, fmt.Sprintf("[%d]", code)) {
			t.Errorf("report output missing [%d]", code)
		}
	}
}

func TestIntegration_TimeoutHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res, output := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "3",
		"-duration", "300ms",
		"-timeout", "50ms",
	})

	if res.Failed == 0 {
		t.Error("expected failures from timeouts")
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected error messages in results")
	}

	if !containsError(res.Errors, "timeout", "deadline") {
		t.Errorf("expected timeout-related error, got: %v", res.Errors)
	}

	if !strings.Contains(output, "Errors:") {
		t.Error("report output missing Errors section")
	}
}

func TestIntegration_ConcurrencyVerification(t *testing.T) {
	var inflight atomic.Int64
	var peak atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := inflight.Add(1)
		defer inflight.Add(-1)

		// Track peak concurrency with lock-free CAS loop.
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	const concurrency = 5
	res, _ := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", fmt.Sprintf("%d", concurrency),
		"-duration", "500ms",
	})

	if res.TotalRequests == 0 {
		t.Fatal("expected requests to be made")
	}

	observed := peak.Load()
	if observed < 2 {
		t.Errorf("peak concurrency = %d, want >= 2", observed)
	}
	if observed > concurrency {
		t.Errorf("peak concurrency = %d, exceeds configured %d", observed, concurrency)
	}
}

func TestIntegration_ShortDuration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		runFullPipeline(t, []string{
			"-url", srv.URL,
			"-concurrency", "3",
			"-duration", "50ms",
		})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test hung: 50ms duration did not complete within 5s")
	}
}

func TestIntegration_HTTPMethods(t *testing.T) {
	tests := []struct {
		method string
	}{
		{method: "POST"},
		{method: "PUT"},
		{method: "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			var receivedMethod atomic.Value

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod.Store(r.Method)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			res, _ := runFullPipeline(t, []string{
				"-url", srv.URL,
				"-method", tt.method,
				"-concurrency", "2",
				"-duration", "100ms",
			})

			if res.TotalRequests == 0 {
				t.Fatal("expected at least one request")
			}

			got, ok := receivedMethod.Load().(string)
			if !ok {
				t.Fatal("server never received a request")
			}
			if got != tt.method {
				t.Errorf("server received method %q, want %q", got, tt.method)
			}
		})
	}
}

func TestIntegration_ConnectionErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Log("server does not support hijacking")
			w.WriteHeader(http.StatusOK)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Logf("hijack failed: %v", err)
			return
		}
		conn.(*net.TCPConn).SetLinger(0)
		conn.Close()
	}))
	defer srv.Close()

	res, output := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "2",
		"-duration", "200ms",
		"-timeout", "1s",
	})

	if res.Failed == 0 {
		t.Error("expected failures from connection errors")
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected error messages in results")
	}

	if !containsError(res.Errors, "eof", "reset", "connection", "broken pipe") {
		t.Errorf("expected connection-related error, got: %v", res.Errors)
	}

	if !strings.Contains(output, "Errors:") {
		t.Error("report output missing Errors section")
	}
}
