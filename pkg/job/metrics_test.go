package job

import (
	"testing"
	"time"
)

func TestMakeMetric(t *testing.T) {
	tests := []struct {
		name   string
		timing string
		value  float64
		job    string
		url    string
		tags   map[string]string
	}{
		{
			name:   "all fields populated",
			timing: "dns_lookup",
			value:  42.5,
			job:    "http_check",
			url:    "https://example.com",
			tags:   map[string]string{"env": "prod", "region": "us-east"},
		},
		{
			name:   "nil tags",
			timing: "connect",
			value:  0,
			job:    "tcp_check",
			url:    "",
			tags:   nil,
		},
		{
			name:   "empty tags",
			timing: "tls_handshake",
			value:  -1.0,
			job:    "ssl_check",
			url:    "https://secure.example.com",
			tags:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			m := MakeMetric(tt.timing, tt.value, tt.job, tt.url, tt.tags)
			after := time.Now()

			if m.Timing != tt.timing {
				t.Errorf("Timing = %q, want %q", m.Timing, tt.timing)
			}
			if m.Value != tt.value {
				t.Errorf("Value = %f, want %f", m.Value, tt.value)
			}
			if m.Job != tt.job {
				t.Errorf("Job = %q, want %q", m.Job, tt.job)
			}
			if m.URL != tt.url {
				t.Errorf("URL = %q, want %q", m.URL, tt.url)
			}
			if m.Timestamp.Before(before) || m.Timestamp.After(after) {
				t.Errorf("Timestamp %v not between %v and %v", m.Timestamp, before, after)
			}
			if m.Timestamp.IsZero() {
				t.Error("Timestamp is zero")
			}
		})
	}
}

func TestMakeEvent(t *testing.T) {
	tests := []struct {
		name       string
		eventName  string
		status     int
		tags       map[string]string
		wantTagLen int
	}{
		{
			name:       "all fields populated",
			eventName:  "site_down",
			status:     503,
			tags:       map[string]string{"env": "prod"},
			wantTagLen: 1,
		},
		{
			name:       "nil tags initializes empty map",
			eventName:  "timeout",
			status:     0,
			tags:       nil,
			wantTagLen: 0,
		},
		{
			name:       "empty tags stays empty map",
			eventName:  "recovered",
			status:     200,
			tags:       map[string]string{},
			wantTagLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			e := MakeEvent(tt.eventName, tt.status, tt.tags)
			after := time.Now()

			if e.Name != tt.eventName {
				t.Errorf("Name = %q, want %q", e.Name, tt.eventName)
			}
			if e.ServerStatus != tt.status {
				t.Errorf("ServerStatus = %d, want %d", e.ServerStatus, tt.status)
			}
			if e.Timestamp.Before(before) || e.Timestamp.After(after) {
				t.Errorf("Timestamp %v not between %v and %v", e.Timestamp, before, after)
			}
			if e.Timestamp.IsZero() {
				t.Error("Timestamp is zero")
			}
			if e.Tags == nil {
				t.Fatal("Tags is nil, expected initialized map")
			}
			if len(e.Tags) != tt.wantTagLen {
				t.Errorf("len(Tags) = %d, want %d", len(e.Tags), tt.wantTagLen)
			}
		})
	}
}
