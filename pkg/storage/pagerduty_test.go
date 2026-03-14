package storage

import (
	"context"
	"testing"
	"time"

	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
)

func TestNewPagerDutyBackend(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.PagerDutyConfig
		wantErr bool
	}{
		{
			name:    "missing routing key",
			cfg:     config.PagerDutyConfig{},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg:  config.PagerDutyConfig{RoutingKey: "test-key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPagerDutyBackend(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPagerDutyBackend() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewPagerDutyBackend_defaults(t *testing.T) {
	b, err := NewPagerDutyBackend(config.PagerDutyConfig{RoutingKey: "test-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if b.config.Namespace != "crabby" {
		t.Errorf("default Namespace = %q, want %q", b.config.Namespace, "crabby")
	}
	if b.config.Client != "crabby" {
		t.Errorf("default Client = %q, want %q", b.config.Client, "crabby")
	}
	if b.config.EventDuration != time.Hour {
		t.Errorf("default EventDuration = %v, want %v", b.config.EventDuration, time.Hour)
	}
	if b.Name() != "pagerduty" {
		t.Errorf("Name() = %q, want %q", b.Name(), "pagerduty")
	}
}

func TestPagerDutyBackend_SendEvent_ignores_non_error(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "200 OK", status: 200},
		{name: "301 redirect", status: 301},
		{name: "399 boundary", status: 399},
		{name: "zero status", status: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewPagerDutyBackend(config.PagerDutyConfig{RoutingKey: "test-key"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			e := job.Event{
				Name:         "test",
				ServerStatus: tt.status,
				Timestamp:    time.Now(),
			}
			// SendEvent should return nil for non-error statuses without
			// attempting to contact PagerDuty.
			if err := b.SendEvent(context.Background(), e); err != nil {
				t.Errorf("SendEvent() returned error for status %d: %v", tt.status, err)
			}
		})
	}
}

func TestPagerDutyBackend_SendEvent_deduplication(t *testing.T) {
	b, err := NewPagerDutyBackend(config.PagerDutyConfig{
		RoutingKey:    "test-key",
		EventDuration: time.Hour,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()

	// Simulate a first event that would set the timestamp in the map.
	// We do this by directly populating eventTimestamps to avoid
	// actually calling the PagerDuty API.
	eventKey := "api-check-500"
	b.eventTimestamps[eventKey] = now

	// An event within the dedup window should be silently dropped.
	e := job.Event{
		Name:         "api-check",
		ServerStatus: 500,
		Timestamp:    now.Add(30 * time.Minute), // within 1h window
	}
	err = b.SendEvent(context.Background(), e)
	if err != nil {
		t.Errorf("expected nil for deduplicated event, got: %v", err)
	}

	// Verify the timestamp was NOT updated (event was dropped).
	if ts := b.eventTimestamps[eventKey]; ts != now {
		t.Errorf("eventTimestamps was updated for deduped event: got %v, want %v", ts, now)
	}
}
