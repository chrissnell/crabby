package storage

import (
	"strings"
	"testing"
	"time"

	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
)

func newTestLogBackend(t *testing.T) *LogBackend {
	t.Helper()
	b, err := NewLogBackend(config.LogConfig{File: "stdout"})
	if err != nil {
		t.Fatalf("NewLogBackend: %v", err)
	}
	return b
}

func TestBuildMetricString(t *testing.T) {
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		metric   job.Metric
		wantSubs []string // substrings expected in output
	}{
		{
			name: "default format with tags",
			metric: job.Metric{
				Job:       "web-check",
				Timing:    "dns_lookup",
				Value:     42.5,
				URL:       "https://example.com",
				Timestamp: ts,
				Tags:      map[string]string{"env": "prod"},
			},
			wantSubs: []string{"[M: web-check]", "dns_lookup:", "42.5", "env: prod"},
		},
		{
			name: "metric with no tags",
			metric: job.Metric{
				Job:       "api",
				Timing:    "connect",
				Value:     10,
				Timestamp: ts,
			},
			wantSubs: []string{"[M: api]", "connect:", "10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestLogBackend(t)
			b.location = time.UTC
			got := b.BuildMetricString(tt.metric)
			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("BuildMetricString() = %q, missing %q", got, sub)
				}
			}
		})
	}
}

func TestBuildEventString(t *testing.T) {
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		event    job.Event
		wantSubs []string
	}{
		{
			name: "default format",
			event: job.Event{
				Name:         "healthcheck",
				ServerStatus: 200,
				Timestamp:    ts,
				Tags:         map[string]string{"region": "us-east"},
			},
			wantSubs: []string{"[E: healthcheck]", "status: 200", "region: us-east"},
		},
		{
			name: "error status no tags",
			event: job.Event{
				Name:         "api-check",
				ServerStatus: 503,
				Timestamp:    ts,
			},
			wantSubs: []string{"[E: api-check]", "status: 503"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestLogBackend(t)
			b.location = time.UTC
			got := b.BuildEventString(tt.event)
			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("BuildEventString() = %q, missing %q", got, sub)
				}
			}
		})
	}
}

func TestBuildTagString(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		want string // exact match for deterministic cases, empty for multi-tag
	}{
		{
			name: "empty tags",
			tags: map[string]string{},
			want: "",
		},
		{
			name: "nil tags",
			tags: nil,
			want: "",
		},
		{
			name: "single tag",
			tags: map[string]string{"env": "prod"},
			want: "env: prod",
		},
		{
			name: "multiple tags contains all",
			tags: map[string]string{"env": "prod", "region": "us"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestLogBackend(t)
			got := b.BuildTagString(tt.tags)

			if tt.want != "" || len(tt.tags) <= 1 {
				if got != tt.want {
					t.Errorf("BuildTagString() = %q, want %q", got, tt.want)
				}
				return
			}

			// For multiple tags, verify all tags are present (map iteration order is random)
			for k, v := range tt.tags {
				sub := k + ": " + v
				if !strings.Contains(got, sub) {
					t.Errorf("BuildTagString() = %q, missing %q", got, sub)
				}
			}
			if !strings.Contains(got, ", ") {
				t.Errorf("BuildTagString() = %q, missing separator", got)
			}
		})
	}
}

func TestNewLogBackend(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.LogConfig
		wantErr bool
	}{
		{
			name: "stdout",
			cfg:  config.LogConfig{File: "stdout"},
		},
		{
			name: "stderr",
			cfg:  config.LogConfig{File: "stderr"},
		},
		{
			name:    "invalid location",
			cfg:     config.LogConfig{File: "stdout", Time: config.TimeConfig{Location: "Invalid/Zone"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewLogBackend(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if b.Name() != "log" {
				t.Errorf("Name() = %q, want %q", b.Name(), "log")
			}
		})
	}
}

func TestNewLogBackend_defaults(t *testing.T) {
	b, err := NewLogBackend(config.LogConfig{File: "stdout"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if b.timeFormat != "2006/01/02 15:04:05" {
		t.Errorf("default timeFormat = %q, want %q", b.timeFormat, "2006/01/02 15:04:05")
	}
	if b.format.Metric == "" {
		t.Error("default metric format should not be empty")
	}
	if b.format.Event == "" {
		t.Error("default event format should not be empty")
	}
	if b.format.Tag == "" {
		t.Error("default tag format should not be empty")
	}
	if b.format.TagSeparator == "" {
		t.Error("default tag separator should not be empty")
	}
}
