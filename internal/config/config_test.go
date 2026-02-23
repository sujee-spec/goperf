package config

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Config
		wantErr bool
	}{
		{
			name: "valid with all flags",
			args: []string{"-url", "http://example.com", "-concurrency", "5", "-duration", "3s", "-method", "POST", "-timeout", "5s"},
			want: Config{
				URL:         "http://example.com",
				Method:      "POST",
				Concurrency: 5,
				Duration:    3 * time.Second,
				Timeout:     5 * time.Second,
			},
		},
		{
			name: "valid with defaults",
			args: []string{"-url", "http://example.com"},
			want: Config{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: 10,
				Duration:    10 * time.Second,
				Timeout:     10 * time.Second,
			},
		},
		{
			name:    "missing url",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "zero concurrency",
			args:    []string{"-url", "http://example.com", "-concurrency", "0"},
			wantErr: true,
		},
		{
			name:    "negative concurrency",
			args:    []string{"-url", "http://example.com", "-concurrency", "-1"},
			wantErr: true,
		},
		{
			name:    "invalid http method",
			args:    []string{"-url", "http://example.com", "-method", "BANANA"},
			wantErr: true,
		},
		{
			name:    "invalid flag",
			args:    []string{"-nonexistent"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
