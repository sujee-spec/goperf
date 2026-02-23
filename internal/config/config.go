package config

import (
	"errors"
	"flag"
	"fmt"
	"time"
)

type Config struct {
	URL         string
	Method      string
	Concurrency int
	Duration    time.Duration
	Timeout     time.Duration
}

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
