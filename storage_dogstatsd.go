package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
)

// DogstatsdConfig describes the YAML-provided configuration for a Datadog
// DogstatsD storage backend
type DogstatsdConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Namespace string `yaml:"metric-namespace,omitempty"`
}

// DogstatsdStorage holds the configuration for a DogstatsD storage backend
type DogstatsdStorage struct {
	DogstatsdConn *statsd.Client
	Namespace     string
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to dogstatsd
func (d DogstatsdStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) chan<- Metric {
	metricChan := make(chan Metric, 10)
	go d.processMetrics(ctx, wg, metricChan)
	return metricChan
}

func (d DogstatsdStorage) processMetrics(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-mchan:
			err := d.SendMetric(m)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request recieved.  Cancelling metrics processor.")
			return
		}
	}
}

// SendMetric sends a metric value to dogstatsd
func (d DogstatsdStorage) SendMetric(m Metric) error {
	var metricName string

	if d.Namespace == "" {
		metricName = fmt.Sprintf("crabby.%v", m.Name)
	} else {
		metricName = fmt.Sprintf("%v.%v", d.Namespace, m.Name)
	}

	err := d.DogstatsdConn.Gauge(metricName, m.Value, nil, 1)
	if err != nil {
		log.Printf("Could not send metric %v: %v\n", m.Name, err)
		return err
	}

	return nil
}

// NewDogstatsdStorage sets up a new Dogstatsd storage backend
func NewDogstatsdStorage(c *Config) DogstatsdStorage {
	var err error
	d := DogstatsdStorage{}

	d.Namespace = c.Storage.Dogstatsd.Namespace

	d.DogstatsdConn, err = statsd.New(fmt.Sprint(c.Storage.Dogstatsd.Host, ":", c.Storage.Dogstatsd.Port))
	if err != nil {
		log.Println("Warning: could not create dogstatsd connection", err)
	}

	return d
}
