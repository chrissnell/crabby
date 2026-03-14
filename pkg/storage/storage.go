package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chrissnell/crabby/pkg/job"
)

// MetricSender is implemented by backends that accept metrics.
type MetricSender interface {
	SendMetric(ctx context.Context, m job.Metric) error
}

// EventSender is implemented by backends that accept events.
type EventSender interface {
	SendEvent(ctx context.Context, e job.Event) error
}

// Backend is the lifecycle interface for storage backends.
type Backend interface {
	Name() string
	Start(ctx context.Context) error
	Close() error
}

// Distributor fans out metrics and events to registered backends.
type Distributor struct {
	backends []Backend
}

// NewDistributor creates a new Distributor.
func NewDistributor() *Distributor {
	return &Distributor{}
}

// AddBackend registers a backend with the distributor.
func (d *Distributor) AddBackend(b Backend) {
	d.backends = append(d.backends, b)
}

// Start starts all registered backends.
func (d *Distributor) Start(ctx context.Context) error {
	for _, b := range d.backends {
		if err := b.Start(ctx); err != nil {
			return fmt.Errorf("starting backend %s: %w", b.Name(), err)
		}
	}
	return nil
}

// Close shuts down all backends.
func (d *Distributor) Close() error {
	var firstErr error
	for _, b := range d.backends {
		if err := b.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SendMetrics fans out metrics to all backends implementing MetricSender.
func (d *Distributor) SendMetrics(ctx context.Context, metrics []job.Metric) {
	for _, b := range d.backends {
		ms, ok := b.(MetricSender)
		if !ok {
			continue
		}
		for _, m := range metrics {
			if err := ms.SendMetric(ctx, m); err != nil {
				slog.Error("sending metric", "backend", b.Name(), "error", err)
			}
		}
	}
}

// SendEvents fans out events to all backends implementing EventSender.
func (d *Distributor) SendEvents(ctx context.Context, events []job.Event) {
	for _, b := range d.backends {
		es, ok := b.(EventSender)
		if !ok {
			continue
		}
		for _, e := range events {
			if err := es.SendEvent(ctx, e); err != nil {
				slog.Error("sending event", "backend", b.Name(), "error", err)
			}
		}
	}
}
