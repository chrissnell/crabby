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

// PrometheusStorage holds the configuration for a Graphite storage backend
type PrometheusStorage struct {
	Namespace string
	Registry  *prometheus.Registry
	url       string
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
			err := p.sendMetric(m)
			if err != nil {
				log.Println(err)
			}
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
		metricName = fmt.Sprintf("crabby.%v.%v", m.Job, m.Timing)
		grouping = push.HostnameGroupingKey()
	} else {
		metricName = fmt.Sprintf("%v.%v.%v", p.Namespace, m.Job, m.Timing)
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
