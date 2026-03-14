package job

import (
	"context"
	"runtime"
	"time"
)

// InternalMetricsJob collects Go runtime metrics.
type InternalMetricsJob struct {
	interval time.Duration
}

// NewInternalMetricsJob creates an internal metrics job.
func NewInternalMetricsJob(intervalSec uint) *InternalMetricsJob {
	d := 15 * time.Second
	if intervalSec > 0 {
		d = time.Duration(intervalSec) * time.Second
	}
	return &InternalMetricsJob{interval: d}
}

func (j *InternalMetricsJob) Name() string            { return "internal_metrics" }
func (j *InternalMetricsJob) Interval() time.Duration { return j.interval }

// Run collects runtime memory and goroutine stats.
func (j *InternalMetricsJob) Run(_ context.Context) ([]Metric, []Event, error) {
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

	mk := func(name string, value float64) Metric {
		return Metric{
			Job:       "internal_metrics",
			Timing:    name,
			Value:     value,
			Timestamp: time.Now(),
		}
	}

	metrics := []Metric{
		mk("mem.alloc", float64(memstats.Alloc)),
		mk("heap.alloc", float64(memstats.HeapAlloc)),
		mk("heap.in_use", float64(memstats.HeapInuse)),
		mk("num_goroutines", float64(runtime.NumGoroutine())),
	}
	return metrics, nil, nil
}
