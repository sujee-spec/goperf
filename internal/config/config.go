// Package config handles CLI flag parsing and validation for goperf.
package config

import (
	"errors"
	"flag"
	"fmt"
	"time"
)

// Config holds the load test parameters parsed from CLI flags.
type Config struct {
	URL         string
	Method      string
	Concurrency int
	Duration    time.Duration
	Timeout     time.Duration
}

var validMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
}

// Parse parses CLI arguments into a Config, returning an error if
// flags are invalid or required values are missing.
func Parse(args []string) (Config, error) {
	fs := flag.NewFlagSet("goperf", flag.ContinueOnError)

	var cfg Config
	fs.StringVar(&cfg.URL, "url", "", "Target URL to test (required)")
	fs.StringVar(&cfg.Method, "method", "GET", "HTTP method")
	fs.IntVar(&cfg.Concurrency, "concurrency", 10, "Number of concurrent workers")
	fs.DurationVar(&cfg.Duration, "duration", 10*time.Second, "Test duration")
	fs.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "Request timeout")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	if !validMethods[c.Method] {
		return fmt.Errorf("unsupported HTTP method %q", c.Method)
	}
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive, got %d", c.Concurrency)
	}
	if c.Duration <= 0 {
		return fmt.Errorf("duration must be positive, got %s", c.Duration)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %s", c.Timeout)
	}
	return nil
}
