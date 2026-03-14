package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
)

// DogstatsdBackend sends metrics and events to Datadog via DogStatsD.
type DogstatsdBackend struct {
	conn      *statsd.Client
	namespace string
}

// NewDogstatsdBackend creates a new DogStatsD backend.
func NewDogstatsdBackend(cfg config.DogstatsdConfig) (*DogstatsdBackend, error) {
	conn, err := statsd.New(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("creating dogstatsd connection: %w", err)
	}
	return &DogstatsdBackend{
		conn:      conn,
		namespace: cfg.Namespace,
	}, nil
}

func (d *DogstatsdBackend) Name() string                  { return "dogstatsd" }
func (d *DogstatsdBackend) Start(_ context.Context) error { return nil }
func (d *DogstatsdBackend) Close() error                  { return d.conn.Close() }

// SendMetric sends a metric to DogStatsD.
func (d *DogstatsdBackend) SendMetric(_ context.Context, m job.Metric) error {
	var metricName string
	if d.namespace == "" {
		metricName = fmt.Sprintf("crabby.%v.%v", m.Job, m.Timing)
	} else {
		metricName = fmt.Sprintf("%v.%v.%v", d.namespace, m.Job, m.Timing)
	}

	tags := MakeDogstatsdTags(m.Tags)
	if err := d.conn.TimeInMilliseconds(metricName, m.Value, tags, 1); err != nil {
		slog.Error("sending dogstatsd metric", "metric", metricName, "error", err)
		return err
	}
	return nil
}

// SendEvent sends a service check to Datadog.
func (d *DogstatsdBackend) SendEvent(_ context.Context, e job.Event) error {
	var eventName string
	if d.namespace == "" {
		eventName = fmt.Sprintf("crabby.%v", e.Name)
	} else {
		eventName = fmt.Sprintf("%v.%v", d.namespace, e.Name)
	}

	sc := &statsd.ServiceCheck{
		Name:    eventName,
		Message: fmt.Sprintf("%v is returning a HTTP status code of %v", e.Name, e.ServerStatus),
	}

	if e.ServerStatus > 0 && e.ServerStatus < 400 {
		sc.Status = statsd.Ok
	} else {
		sc.Status = statsd.Critical
	}

	sc.Tags = MakeDogstatsdTags(e.Tags)
	return d.conn.ServiceCheck(sc)
}

// MakeDogstatsdTags converts a tag map to DogStatsD tag format (key:value).
func MakeDogstatsdTags(tags map[string]string) []string {
	var dogTags []string
	for k, v := range tags {
		dogTags = append(dogTags, fmt.Sprintf("%v:%v", k, v))
	}
	return dogTags
}
