package job

import (
	"context"
	"testing"
	"time"
)

func TestInternalMetricsJobName(t *testing.T) {
	j := NewInternalMetricsJob(0)
	if got := j.Name(); got != "internal_metrics" {
		t.Errorf("Name() = %q, want %q", got, "internal_metrics")
	}
}

func TestInternalMetricsJobInterval(t *testing.T) {
	tests := []struct {
		name        string
		intervalSec uint
		want        time.Duration
	}{
		{
			name:        "default interval when zero",
			intervalSec: 0,
			want:        15 * time.Second,
		},
		{
			name:        "custom interval 30s",
			intervalSec: 30,
			want:        30 * time.Second,
		},
		{
			name:        "custom interval 1s",
			intervalSec: 1,
			want:        1 * time.Second,
		},
		{
			name:        "large interval",
			intervalSec: 3600,
			want:        3600 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := NewInternalMetricsJob(tt.intervalSec)
			if got := j.Interval(); got != tt.want {
				t.Errorf("Interval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInternalMetricsJobRun(t *testing.T) {
	j := NewInternalMetricsJob(10)
	metrics, events, err := j.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if events != nil {
		t.Errorf("Run() events = %v, want nil", events)
	}

	wantTimings := []string{"mem.alloc", "heap.alloc", "heap.in_use", "num_goroutines"}

	if len(metrics) != len(wantTimings) {
		t.Fatalf("Run() returned %d metrics, want %d", len(metrics), len(wantTimings))
	}

	for i, wantTiming := range wantTimings {
		t.Run(wantTiming, func(t *testing.T) {
			m := metrics[i]
			if m.Timing != wantTiming {
				t.Errorf("metrics[%d].Timing = %q, want %q", i, m.Timing, wantTiming)
			}
			if m.Job != "internal_metrics" {
				t.Errorf("metrics[%d].Job = %q, want %q", i, m.Job, "internal_metrics")
			}
			if m.Timestamp.IsZero() {
				t.Errorf("metrics[%d].Timestamp is zero", i)
			}
			if m.Value < 0 {
				t.Errorf("metrics[%d].Value = %f, want >= 0", i, m.Value)
			}
		})
	}
}
