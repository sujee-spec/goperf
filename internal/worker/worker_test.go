package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunSendsResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	results := make(chan Result, 100)
	client := srv.Client()

	// Drain results concurrently to avoid blocking the worker
	var collected []Result
	done := make(chan struct{})
	go func() {
		for r := range results {
			collected = append(collected, r)
		}
		close(done)
	}()

	Run(ctx, client, "GET", srv.URL, results)
	close(results)
	<-done

	if len(collected) == 0 {
		t.Fatal("expected at least one result")
	}

	var succeeded int
	for _, r := range collected {
		// The last request may fail due to context cancellation — that's expected
		if r.Error != nil {
			continue
		}
		if r.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", r.StatusCode)
		}
		if r.Duration <= 0 {
			t.Error("expected positive duration")
		}
		succeeded++
	}

	if succeeded == 0 {
		t.Fatal("expected at least one successful result")
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan Result, 1000)
	client := srv.Client()

	doneCh := make(chan struct{})
	go func() {
		Run(ctx, client, "GET", srv.URL, results)
		close(doneCh)
	}()

	// Let it run briefly then cancel
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
}

func TestRunRecordsErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results := make(chan Result, 100)
	client := srv.Client()

	var collected []Result
	done := make(chan struct{})
	go func() {
		for r := range results {
			collected = append(collected, r)
		}
		close(done)
	}()

	Run(ctx, client, "GET", srv.URL, results)
	close(results)
	<-done

	if len(collected) == 0 {
		t.Fatal("expected at least one result")
	}

	var got500 int
	for _, r := range collected {
		if r.Error != nil {
			continue // context cancellation at shutdown is expected
		}
		if r.StatusCode != 500 {
			t.Errorf("expected status 500, got %d", r.StatusCode)
		}
		got500++
	}

	if got500 == 0 {
		t.Fatal("expected at least one 500 response")
	}
}
