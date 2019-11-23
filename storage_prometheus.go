package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusConfig describes the YAML-provided configuration for a Prometheus
// pushgateway storage backend
type PrometheusConfig struct {
	ListenAddr string `yaml:"listen-addr"`
	Namespace  string `yaml:"metric-namespace,omitempty"`
}

// PrometheusStorage holds the configuration for a Graphite storage backend
type PrometheusStorage struct {
	ListenAddr        string
	Namespace         string
	Registry          *prometheus.Registry
	RegisteredMetrics map[string]*prometheus.GaugeVec
}

// NewPrometheusStorage sets up a new Prometheus storage backend
func NewPrometheusStorage(c *Config) PrometheusStorage {
	p := PrometheusStorage{}

	p.RegisteredMetrics = make(map[string]*prometheus.GaugeVec)

	p.Namespace = strings.ReplaceAll(c.Storage.Prometheus.Namespace, "-", "_")

	p.ListenAddr = c.Storage.Prometheus.ListenAddr

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

	logger := log.New(os.Stdout, "", log.LstdFlags)

	server := &http.Server{
		Addr:         p.ListenAddr,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	http.Handle("/metrics", promhttp.HandlerFor(p.Registry, promhttp.HandlerOpts{
		ErrorLog: log.New(os.Stdout, "", log.LstdFlags),
	}))

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		err := http.ListenAndServe(p.ListenAddr, nil)
		if err != nil {
			log.Fatalf("unable to start Prometheus listener: %s\n", err)
		}
	}()

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				log.Println("Cancellation request received.  Cancelling job runner.")
				server.Shutdown(ctx)
				return
			}
		}
	}(ctx)

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

	if p.Namespace == "" {
		metricName = fmt.Sprintf("crabby_%v", m.Timing)
	} else {
		metricName = fmt.Sprintf("%v_%v", p.Namespace, m.Timing)
	}

	m.Tags["crabby_job"] = m.Job
	m.Tags["url"] = m.URL

	metricName = strings.ReplaceAll(metricName, ".", "_")

	promLabelNames := makePrometheusLabelSliceFromTagsMap(m.Tags)

	_, present := p.RegisteredMetrics[metricName]

	// If this metric vector is present in our map of metrics, we'll fetch the gauge and set the current value
	if present {
		metric, err := p.RegisteredMetrics[metricName].GetMetricWith(m.Tags)
		if err != nil {
			return fmt.Errorf("unable to get metric %v with tags %+v: %v", metricName, m.Tags, err)
		}
		metric.Set(m.Value)
	} else {
		// The metric wasn't present in our map, so we'll set up a new gauge vector
		p.RegisteredMetrics[metricName] = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: "Crabby timing metric, in milliseconds",
			},
			promLabelNames,
		)
		p.Registry.MustRegister(p.RegisteredMetrics[metricName])
	}

	return nil
}

// sendEvent is necessary to implement the StorageEngine interface.
func (p PrometheusStorage) sendEvent(e Event) error {
	var err error
	return err
}

func makePrometheusLabelSliceFromTagsMap(tags map[string]string) []string {
	var promLabels []string

	for k := range tags {
		promLabels = append(promLabels, k)
	}

	return promLabels
}
