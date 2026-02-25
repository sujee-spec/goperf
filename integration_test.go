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

func runFullPipeline(t *testing.T, args []string) (engine.Result, string, error) {
	t.Helper()

	cfg, err := config.Parse(args)
	if err != nil {
		return engine.Result{}, "", fmt.Errorf("config parse: %w", err)
	}

	res := engine.Run(cfg)

	var buf bytes.Buffer
	if err := report.Print(&buf, cfg, res); err != nil {
		return res, "", fmt.Errorf("report print: %w", err)
	}

	return res, buf.String(), nil
}

func TestIntegration_FullSuccessfulRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res, output, err := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "5",
		"-duration", "200ms",
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if res.TotalRequests == 0 {
		t.Fatal("expected at least one request")
	}
	if res.Succeeded == 0 {
		t.Error("expected some successful requests")
	}
	// A small number of in-flight requests may fail when the context expires.
	// Allow up to 1% failure rate (minimum 2) to account for this.
	maxAllowed := res.TotalRequests / 100
	if maxAllowed < 2 {
		maxAllowed = 2
	}
	if res.Failed > maxAllowed {
		t.Errorf("too many failures: %d out of %d (max allowed %d)",
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
			t.Errorf("report missing section %q", section)
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

	res, output, err := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "2",
		"-duration", "200ms",
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if res.Succeeded == 0 {
		t.Error("expected some successes")
	}
	if res.Failed == 0 {
		t.Error("expected some failures")
	}
	if res.Succeeded+res.Failed != res.TotalRequests {
		t.Errorf("succeeded (%d) + failed (%d) != total (%d)",
			res.Succeeded, res.Failed, res.TotalRequests)
	}

	if _, ok := res.StatusCodes[200]; !ok {
		t.Error("expected status code 200 in results")
	}
	if _, ok := res.StatusCodes[500]; !ok {
		t.Error("expected status code 500 in results")
	}

	if !strings.Contains(output, "[200]") {
		t.Error("report missing [200] status code")
	}
	if !strings.Contains(output, "[500]") {
		t.Error("report missing [500] status code")
	}
}

func TestIntegration_TimeœoutHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res, output, err := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "3",
		"-duration", "300ms",
		"-timeout", "50ms",
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if res.Failed == 0 {
		t.Error("expected failures from timeouts")
	}
	if len(res.Errors) == 0 {
		t.Error("expected error messages in results")
	}

	hasTimeoutErr := false
	for msg := range res.Errors {
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline") {
			hasTimeoutErr = true
			break
		}
	}
	if !hasTimeoutErr {
		t.Errorf("expected timeout-related error, got errors: %v", res.Errors)
	}

	if !strings.Contains(output, "Errors:") {
		t.Error("report missing Errors section")
	}
}

func TestIntegration_ConcurrencyVerification(t *testing.T) {
	var inflight atomic.Int64
	var peak atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := inflight.Add(1)
		defer inflight.Add(-1)

		// Track peak concurrency.
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

	concurrency := 5
	res, _, err := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", fmt.Sprintf("%d", concurrency),
		"-duration", "500ms",
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if res.TotalRequests == 0 {
		t.Fatal("expected requests to be made")
	}

	observed := peak.Load()
	if observed < 2 {
		t.Errorf("expected peak concurrency >= 2, got %d", observed)
	}
	if observed > int64(concurrency) {
		t.Errorf("peak concurrency %d exceeded configured %d", observed, concurrency)
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
		_, _, err := runFullPipeline(t, []string{
			"-url", srv.URL,
			"-concurrency", "3",
			"-duration", "50ms",
		})
		if err != nil {
			t.Errorf("pipeline failed: %v", err)
		}
	}()

	select {
	case <-done:
		// Completed normally.
	case <-time.After(5 * time.Second):
		t.Fatal("test hung: 50ms duration did not complete within 5s")
	}
}

func TestIntegration_HTTPMethods(t *testing.T) {
	methods := []string{"POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var receivedMethod atomic.Value

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod.Store(r.Method)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			res, _, err := runFullPipeline(t, []string{
				"-url", srv.URL,
				"-method", method,
				"-concurrency", "2",
				"-duration", "100ms",
			})
			if err != nil {
				t.Fatalf("pipeline failed: %v", err)
			}

			if res.TotalRequests == 0 {
				t.Fatal("expected at least one request")
			}

			got, ok := receivedMethod.Load().(string)
			if !ok {
				t.Fatal("server never received a request")
			}
			if got != method {
				t.Errorf("server received method %q, want %q", got, method)
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

	res, output, err := runFullPipeline(t, []string{
		"-url", srv.URL,
		"-concurrency", "2",
		"-duration", "200ms",
		"-timeout", "1s",
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if res.Failed == 0 {
		t.Error("expected failures from connection errors")
	}
	if len(res.Errors) == 0 {
		t.Error("expected error messages in results")
	}

	hasConnErr := false
	for msg := range res.Errors {
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "eof") ||
			strings.Contains(lower, "reset") ||
			strings.Contains(lower, "connection") ||
			strings.Contains(lower, "broken pipe") {
			hasConnErr = true
			break
		}
	}
	if !hasConnErr {
		t.Errorf("expected connection-related error, got errors: %v", res.Errors)
	}

	if !strings.Contains(output, "Errors:") {
		t.Error("report missing Errors section")
	}
}
