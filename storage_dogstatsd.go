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
	Host      string            `yaml:"host"`
	Port      int               `yaml:"port"`
	Namespace string            `yaml:"metric-namespace"`
	Tags      map[string]string `yaml:"tags,omitempty"`
}

// DogstatsdStorage holds the configuration for a DogstatsD storage backend
type DogstatsdStorage struct {
	DogstatsdConn *statsd.Client
	Namespace     string
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to dogstatsd
func (d DogstatsdStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	// Datadog storage supports both metrics and events, so we'll initialize both channels
	metricChan := make(chan Metric, 10)
	eventChan := make(chan Event, 10)

	go d.processMetricsAndEvents(ctx, wg, metricChan, eventChan)

	return metricChan, eventChan
}

func (d DogstatsdStorage) processMetricsAndEvents(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric, echan <-chan Event) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-mchan:
			err := d.sendMetric(m)
			if err != nil {
				log.Println(err)
			}
		case e := <-echan:
			err := d.sendEvent(e)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request recieved.  Cancelling metrics processor.")
			return
		}
	}
}

// sendMetric sends a metric value to dogstatsd
func (d DogstatsdStorage) sendMetric(m Metric) error {
	var metricName string

	if d.Namespace == "" {
		metricName = fmt.Sprintf("crabby.%v.%v", m.Name, m.Timing)
	} else {
		metricName = fmt.Sprintf("%v.%v.%v", d.Namespace, m.Name, m.Timing)
	}

	// Add all of our metric tags to the metric payload
	for k, v := range m.Tags {
		d.DogstatsdConn.Tags = append(d.DogstatsdConn.Tags, k+":"+v)
	}

	err := d.DogstatsdConn.TimeInMilliseconds(metricName, m.Value, nil, 1)
	if err != nil {
		log.Printf("Could not send metric %v.%v: %v\n", m.Name, m.Timing, err)
		return err
	}

	return nil
}

// sendEvent sends an event (as a service check) to the Datadog API endpoint
func (d DogstatsdStorage) sendEvent(e Event) error {

	var eventName string

	if d.Namespace == "" {
		eventName = fmt.Sprintf("crabby.%v", e.Name)
	} else {
		eventName = fmt.Sprintf("%v.%v", d.Namespace, e.Name)
	}

	// While Crabby calls this an "event", it's really a "service check" in
	// Datadog parlance.  Datadog does have the concept of "events" but it's
	// more difficult to set up monitoring for events than it is service checks.
	// With service checks, we can send the status with every check.  We just
	// set the Status field to indicate whether things are OK (response code 1xx/2xx/3xx)
	// or are failing (response code 4xx/5xx)
	sc := &statsd.ServiceCheck{
		Name:    eventName,
		Message: fmt.Sprintf("%v is returning a HTTP status code of %v", e.Name, e.ServerStatus),
	}

	// Add all of our metric tags to the metric payload
	for k, v := range e.Tags {
		d.DogstatsdConn.Tags = append(d.DogstatsdConn.Tags, k+":"+v)
	}

	if (e.ServerStatus < 400) && (e.ServerStatus > 0) {
		sc.Status = statsd.Ok
	} else {
		sc.Status = statsd.Critical
	}

	err := d.DogstatsdConn.ServiceCheck(sc)

	return err
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

	for k, v := range c.Storage.Dogstatsd.Tags {
		d.DogstatsdConn.Tags = append(d.DogstatsdConn.Tags, k+":"+v)
	}
	return d
}
