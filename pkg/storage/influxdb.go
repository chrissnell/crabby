package storage

import (
	"context"
	"fmt"
	"log/slog"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"

	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
)

// InfluxDBBackend sends metrics to InfluxDB v2.
type InfluxDBBackend struct {
	client    influxdb2.Client
	writeAPI  api.WriteAPIBlocking
	namespace string
}

// NewInfluxDBBackend creates a new InfluxDB v2 backend.
func NewInfluxDBBackend(cfg config.InfluxDBConfig) (*InfluxDBBackend, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("missing influxdb host")
	}

	client := influxdb2.NewClient(cfg.Host, cfg.Token)
	writeAPI := client.WriteAPIBlocking(cfg.Org, cfg.Bucket)

	return &InfluxDBBackend{
		client:    client,
		writeAPI:  writeAPI,
		namespace: cfg.Namespace,
	}, nil
}

func (i *InfluxDBBackend) Name() string                  { return "influxdb" }
func (i *InfluxDBBackend) Start(_ context.Context) error { return nil }
func (i *InfluxDBBackend) Close() error                  { i.client.Close(); return nil }

// SendMetric sends a metric to InfluxDB v2.
func (i *InfluxDBBackend) SendMetric(ctx context.Context, m job.Metric) error {
	var metricName string
	if i.namespace == "" {
		metricName = fmt.Sprintf("crabby.%v", m.Job)
	} else {
		metricName = fmt.Sprintf("%v.%v", i.namespace, m.Job)
	}

	p := influxdb2.NewPoint(metricName, m.Tags, map[string]interface{}{m.Timing: m.Value}, m.Timestamp)
	if err := i.writeAPI.WritePoint(ctx, p); err != nil {
		slog.Error("writing to influxdb", "error", err)
		return fmt.Errorf("writing data point: %w", err)
	}
	return nil
}
