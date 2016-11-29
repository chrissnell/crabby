package main

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"
)

// startInternalMetrics collects metrics from Crabby's Go runtime and reports them as metrics
func startInternalMetrics(ctx context.Context, wg *sync.WaitGroup, storage *Storage) {
	var memstats runtime.MemStats
	metricsTicker := time.NewTicker(15 * time.Second)

	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-metricsTicker.C:
			runtime.ReadMemStats(&memstats)
			storage.MetricDistributor <- makeInternalMetric("crabby-process.mem.alloc", float64(memstats.Alloc))
			storage.MetricDistributor <- makeInternalMetric("crabby-process.heap.alloc", float64(memstats.HeapAlloc))
			storage.MetricDistributor <- makeInternalMetric("crabby-process.heap.in_use", float64(memstats.HeapInuse))
			storage.MetricDistributor <- makeInternalMetric("crabby-process.num_goroutines", float64(runtime.NumGoroutine()))
		case <-ctx.Done():
			log.Println("Cancellation request received.  Cancelling job runner.")
			return
		}
	}

}

func makeInternalMetric(name string, value float64) Metric {
	m := Metric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
	}
	return m
}
