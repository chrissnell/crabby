package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chrissnell/crabby/pkg/config"
)

func TestHECEvent_JSON(t *testing.T) {
	tests := []struct {
		name  string
		event HECEvent
		check func(t *testing.T, data map[string]interface{})
	}{
		{
			name: "basic fields",
			event: HECEvent{
				Time:       1718451000000,
				Host:       "myhost",
				Source:     "crabby",
				SourceType: "metric",
				Index:      "main",
				Event:      map[string]string{"key": "val"},
			},
			check: func(t *testing.T, data map[string]interface{}) {
				if data["host"] != "myhost" {
					t.Errorf("host = %v, want myhost", data["host"])
				}
				if data["sourcetype"] != "metric" {
					t.Errorf("sourcetype = %v, want metric", data["sourcetype"])
				}
				if data["index"] != "main" {
					t.Errorf("index = %v, want main", data["index"])
				}
				if data["source"] != "crabby" {
					t.Errorf("source = %v, want crabby", data["source"])
				}
				// JSON numbers decode as float64
				if data["time"] != float64(1718451000000) {
					t.Errorf("time = %v, want 1718451000000", data["time"])
				}
			},
		},
		{
			name: "nested event data",
			event: HECEvent{
				Time:       0,
				SourceType: "event",
				Index:      "events",
				Event:      map[string]interface{}{"status": 200, "name": "check"},
			},
			check: func(t *testing.T, data map[string]interface{}) {
				ev, ok := data["event"].(map[string]interface{})
				if !ok {
					t.Fatalf("event field not a map: %T", data["event"])
				}
				if ev["name"] != "check" {
					t.Errorf("event.name = %v, want check", ev["name"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}

			var decoded map[string]interface{}
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}
			tt.check(t, decoded)
		})
	}
}

func TestNewSplunkHECBackend_default_timeout(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		wantTimeout    time.Duration
	}{
		{
			name:        "zero defaults to 15s",
			timeout:     0,
			wantTimeout: 15 * time.Second,
		},
		{
			name:        "explicit timeout preserved",
			timeout:     30 * time.Second,
			wantTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewSplunkHECBackend(config.SplunkHecConfig{}, tt.timeout)
			if err != nil {
				t.Fatalf("NewSplunkHECBackend: %v", err)
			}
			if b.client.Timeout != tt.wantTimeout {
				t.Errorf("client.Timeout = %v, want %v", b.client.Timeout, tt.wantTimeout)
			}
			if b.Name() != "splunk_hec" {
				t.Errorf("Name() = %q, want %q", b.Name(), "splunk_hec")
			}
		})
	}
}
