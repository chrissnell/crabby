package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Metric holds one metric data point
type Metric struct {
	Name      string
	Value     float64
	Tags      map[string]string
	Timestamp time.Time
}

// Event holds one monitoring event
type Event struct {
	Name         string
	ServerStatus int
	Tags         map[string]string
	Timestamp    time.Time
}

// Storage holds our active storage backends
type Storage struct {
	Engines           []StorageEngine
	MetricDistributor chan Metric
	EventDistributor  chan Event
}

// StorageEngine holds a backend storage engine's interface as well as
// a channel for passing metrics to the engine
type StorageEngine struct {
	I              StorageEngineInterface
	M              chan<- Metric
	E              chan<- Event
	AcceptsMetrics bool
	AcceptsEvents  bool
}

// StorageEngineInterface is an interface that provides a few standardized
// methods for various storage backends
type StorageEngineInterface interface {
	sendMetric(Metric) error
	sendEvent(Event) error
	StartStorageEngine(context.Context, *sync.WaitGroup) (chan<- Metric, chan<- Event)
}

// NewStorage creats a Storage object, populated with all configured
// StorageEngines
func NewStorage(ctx context.Context, wg *sync.WaitGroup, c *Config) (*Storage, error) {
	var err error

	s := Storage{}

	// Initialize our channel for passing metrics to the StorageDistributor
	s.MetricDistributor = make(chan Metric, 20)

	// Initialize our channel for passing events to the StorageDistributor
	s.EventDistributor = make(chan Event, 20)

	// Check the configuration file for various supported storage backends
	// and enable them if found
	if c.Storage.Graphite.Host != "" {
		err = s.AddEngine(ctx, wg, "graphite", c)
		if err != nil {
			return &s, fmt.Errorf("could not add Graphite storage backend: %v", err)
		}
	}

	if c.Storage.Dogstatsd.Host != "" {
		err = s.AddEngine(ctx, wg, "dogstatsd", c)
		if err != nil {
			return &s, fmt.Errorf("could not add dogstatsd storage backend: %v", err)
		}
	}

	if c.Storage.Prometheus.Host != "" {
		err = s.AddEngine(ctx, wg, "prometheus", c)
		if err != nil {
			return &s, fmt.Errorf("could not start Prometheus storage backend: %v", err)
		}
	}

	if c.Storage.Riemann.Host != "" {
		err = s.AddEngine(ctx, wg, "riemann", c)
		if err != nil {
			return &s, fmt.Errorf("could not start Riemann storage backend: %v", err)
		}
	}

	// Start our storage distributor to distribute received metrics and events
	// to storage backends
	go s.storageDistributor(ctx, wg)

	return &s, nil
}

// AddEngine adds a new StorageEngine of name engineName to our Storage object
func (s *Storage) AddEngine(ctx context.Context, wg *sync.WaitGroup, engineName string, c *Config) error {
	var err error

	switch engineName {
	case "graphite":
		se := StorageEngine{}
		se.I = NewGraphiteStorage(c)
		se.AcceptsEvents = false
		se.AcceptsMetrics = true
		se.M, se.E = se.I.StartStorageEngine(ctx, wg)
		s.Engines = append(s.Engines, se)
	case "dogstatsd":
		se := StorageEngine{}
		se.I = NewDogstatsdStorage(c)
		se.AcceptsEvents = true
		se.AcceptsMetrics = true
		se.M, se.E = se.I.StartStorageEngine(ctx, wg)
		s.Engines = append(s.Engines, se)
	case "prometheus":
		se := StorageEngine{}
		se.I = NewPrometheusStorage(c)
		se.AcceptsEvents = false
		se.AcceptsMetrics = true
		se.M, se.E = se.I.StartStorageEngine(ctx, wg)
		s.Engines = append(s.Engines, se)
	case "riemann":
		se := StorageEngine{}
		se.I, err = NewRiemannStorage(c)
		if err != nil {
			log.Fatalln("Could not connect to Riemann storage backend:", err)
		}
		se.AcceptsEvents = true
		se.AcceptsMetrics = true
		se.M, se.E = se.I.StartStorageEngine(ctx, wg)
		s.Engines = append(s.Engines, se)
	}

	return nil
}

// storageDistributor receives metrics from gathers and fans them out to the various
// storage backends
func (s *Storage) storageDistributor(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case e := <-s.EventDistributor:
			for _, en := range s.Engines {
				// We only forward events onward if the engine supports events
				if en.AcceptsEvents {
					en.E <- e
				}
			}
		case m := <-s.MetricDistributor:
			for _, en := range s.Engines {
				// We only forward metrics onwards if the engine supports metrics
				if en.AcceptsMetrics {
					en.M <- m
				}
			}
		case <-ctx.Done():
			log.Println("Cancellation request received.  Cancelling metric distributor.")
			return nil
		}
	}
}

// makeMetric creates a Metric from raw values and metric names
func makeMetric(j Job, timing string, value float64) Metric {

	m := Metric{
		Name:      fmt.Sprintf("%v.%v", j.Name, timing),
		Value:     value,
		Timestamp: time.Now(),
	}

	for k, v := range j.Tags {
		m.Tags[k] = v
	}

	return m
}

// makeEvent creates an Event from raw values and event names
func makeEvent(j Job, status int) Event {

	e := Event{
		Name:         j.Name,
		ServerStatus: status,
		Timestamp:    time.Now(),
	}

	for k, v := range j.Tags {
		e.Tags[k] = v
	}

	return e
}
