package storage

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusBackend exposes metrics via a Prometheus endpoint.
type PrometheusBackend struct {
	listenAddr        string
	namespace         string
	registry          *prometheus.Registry
	registeredMetrics map[string]*prometheus.GaugeVec
	server            *http.Server
}

// NewPrometheusBackend creates a new Prometheus backend.
func NewPrometheusBackend(cfg config.PrometheusConfig) *PrometheusBackend {
	return &PrometheusBackend{
		listenAddr:        cfg.ListenAddr,
		namespace:         strings.ReplaceAll(cfg.Namespace, "-", "_"),
		registry:          prometheus.NewRegistry(),
		registeredMetrics: make(map[string]*prometheus.GaugeVec),
	}
}

func (p *PrometheusBackend) Name() string { return "prometheus" }

// Start starts the HTTP server for Prometheus scraping.
func (p *PrometheusBackend) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}))

	p.server = &http.Server{
		Addr:    p.listenAddr,
		Handler: mux,
	}

	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("prometheus listener failed", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		p.server.Shutdown(context.Background())
	}()

	return nil
}

// Close shuts down the HTTP server.
func (p *PrometheusBackend) Close() error {
	if p.server != nil {
		return p.server.Shutdown(context.Background())
	}
	return nil
}

// SendMetric records a metric value for Prometheus scraping.
func (p *PrometheusBackend) SendMetric(_ context.Context, m job.Metric) error {
	var metricName string
	if p.namespace == "" {
		metricName = fmt.Sprintf("crabby_%v", m.Timing)
	} else {
		metricName = fmt.Sprintf("%v_%v", p.namespace, m.Timing)
	}

	if m.Tags == nil {
		m.Tags = make(map[string]string)
	}
	m.Tags["crabby_job"] = m.Job
	m.Tags["url"] = m.URL

	metricName = strings.ReplaceAll(metricName, ".", "_")
	labelNames := MakePrometheusLabels(m.Tags)

	if gv, present := p.registeredMetrics[metricName]; present {
		metric, err := gv.GetMetricWith(m.Tags)
		if err != nil {
			return fmt.Errorf("getting metric %v with tags %+v: %w", metricName, m.Tags, err)
		}
		metric.Set(m.Value)
	} else {
		p.registeredMetrics[metricName] = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: "Crabby timing metric, in milliseconds",
			},
			labelNames,
		)
		p.registry.MustRegister(p.registeredMetrics[metricName])
	}

	return nil
}

// MakePrometheusLabels extracts label names from a tag map.
func MakePrometheusLabels(tags map[string]string) []string {
	var labels []string
	for k := range tags {
		labels = append(labels, k)
	}
	return labels
}
