package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkRun(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := srv.Client()
	results := make(chan Result, 1000)

	// Drain results in the background.
	go func() {
		for range results {
		}
	}()

	b.ResetTimer()
	for b.Loop() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		Run(ctx, client, "GET", srv.URL, results)
		cancel()
	}
	b.StopTimer()
	close(results)
}

func BenchmarkRunLargeResponse(b *testing.B) {
	body := make([]byte, 64*1024) // 64 KB response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer srv.Close()

	client := srv.Client()
	results := make(chan Result, 1000)

	go func() {
		for range results {
		}
	}()

	b.ResetTimer()
	for b.Loop() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		Run(ctx, client, "GET", srv.URL, results)
		cancel()
	}
	b.StopTimer()
	close(results)
}
