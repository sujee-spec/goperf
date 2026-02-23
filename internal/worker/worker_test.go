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

	for _, r := range collected {
		if r.Error != nil {
			t.Errorf("unexpected error: %v", r.Error)
		}
		if r.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", r.StatusCode)
		}
		if r.Duration <= 0 {
			t.Error("expected positive duration")
		}
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

	for _, r := range collected {
		if r.StatusCode != 500 {
			t.Errorf("expected status 500, got %d", r.StatusCode)
		}
	}
}
