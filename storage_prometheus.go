package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// PrometheusConfig describes the YAML-provided configuration for a Prometheus
// pushgateway storage backend
type PrometheusConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Namespace string `yaml:"metric-namespace,omitempty"`
}

// CurrentMetrics is a map of Metrics + a mutex that maintains the most recent metrics
// that we are collecting.  These are maintained so that a Prometheus client can connect
// to the built-in Prometheus endpoint and fetch the very latest result for this
// metric.
type CurrentMetrics struct {
	metrics map[string]MetricStore
	sync.RWMutex
}

// MetricStore holds a Prometheus GaugeVec for a given metric
type MetricStore struct {
	name     string
	timing   string
	gaugeVec prometheus.GaugeVec
}

// PrometheusStorage holds the configuration for a Graphite storage backend
type PrometheusStorage struct {
	Namespace      string
	Registry       *prometheus.Registry
	url            string
	currentMetrics *CurrentMetrics
}

// NewPrometheusStorage sets up a new Prometheus storage backend
func NewPrometheusStorage(c *Config) PrometheusStorage {
	p := PrometheusStorage{}

	p.Namespace = c.Storage.Prometheus.Namespace
	p.url = fmt.Sprint(c.Storage.Prometheus.Host, ":", c.Storage.Prometheus.Port)

	p.Registry = prometheus.NewRegistry()

	return p
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to a Prometheus pushgateway
func (p PrometheusStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	// We're going to declare eventChan here but not initialize the channel because Prometheus
	// storage doesn't support Events, only Metrics.  We should never receive an Event on this
	// channel and if something mistakenly sends one, the program will panic.
	var eventChan chan<- Event

	// We *do* support Metrics, so we'll initialize this as a buffered channel
	metricChan := make(chan Metric, 10)

	// Start processing the metrics we receive
	go p.processMetrics(ctx, wg, metricChan)

	return metricChan, eventChan
}

func (p PrometheusStorage) processMetrics(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-mchan:
			p.storeCurrentMetric(m)
		case <-ctx.Done():
			log.Println("Cancellation request recieved.  Cancelling metrics processor.")
			return
		}
	}
}

// sendMetric sends a metric value to Prometheus
func (p PrometheusStorage) sendMetric(m Metric) error {
	var metricName string

	grouping := make(map[string]string)

	if p.Namespace == "" {
		metricName = fmt.Sprintf("crabby.%v", m.Name)
		grouping = push.HostnameGroupingKey()
	} else {
		metricName = fmt.Sprintf("%v.%v", p.Namespace, m.Name)
		grouping["crabby"] = p.Namespace
	}

	pm := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: metricName,
		Help: "Crabby timing metric, in milliseconds",
	})

	pm.Set(m.Value)

	p.Registry.MustRegister(pm)

	push.AddFromGatherer("crabby", grouping, p.url, p.Registry)

	return nil
}

// sendEvent is necessary to implement the StorageEngine interface.
func (p PrometheusStorage) sendEvent(e Event) error {
	var err error
	return err
}

func (p PrometheusStorage) storeCurrentMetric(m Metric) {
	p.currentMetrics.Lock()
	defer p.currentMetrics.Unlock()

	// See if this metric name exists in the current metrics map
	cm, exists := p.currentMetrics.metrics[m.Timing]

	// If the metric doesn't exist, create a new MetricsStore and Gauge for it
	if !exists {
		var glabel []string

		for k := range m.Tags {
			glabel = append(glabel, k)
		}

		gauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: m.Timing,
			}, glabel,
		)

		// Create a MetricsStore/Gauge
		ms := MetricStore{
			name:     m.Name,
			gaugeVec: *gauge,
		}

		// ...and add that to our map
		p.currentMetrics.metrics[m.Name] = ms

		// Register the Gauge with Prometheus
		prometheus.MustRegister(p.currentMetrics.metrics[m.Timing].gaugeVec)

	} else {
		labels := prometheus.Labels{}
		for k, v := range m.Tags {
			labels[k] = v
		}
		labels["name"] = m.Name
		cm.gaugeVec.With(labels).Set(m.Value)
	}

}
