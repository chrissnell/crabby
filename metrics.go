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
	Timestamp time.Time
}

// Storage holds our active storage backends
type Storage struct {
	Engines           []StorageEngine
	MetricDistributor chan Metric
}

// StorageEngine holds a backend storage engine's interface as well as
// a channel for passing metrics to the engine
type StorageEngine struct {
	I StorageEngineInterface
	C chan<- Metric
}

// StorageEngineInterface is an interface that provides a few standardized
// methods for various storage backends
type StorageEngineInterface interface {
	SendMetric(Metric) error
	StartStorageEngine(context.Context, *sync.WaitGroup) chan<- Metric
}

// NewStorage creats a Storage object, populated with all configured
// StorageEngines
func NewStorage(ctx context.Context, wg *sync.WaitGroup, c *Config) (*Storage, error) {
	var err error

	s := Storage{}

	// Initialize our channel for passing metrics to the MetricDistributor
	s.MetricDistributor = make(chan Metric, 20)

	// Start our metric distributor to distribute received metrics to storage
	// backends
	go s.metricDistributor(ctx, wg)

	// Check the configuration file for various supported storage backends
	// and enable them if found
	if c.Storage.Graphite.Host != "" {
		err = s.AddEngine(ctx, wg, "graphite", c)
		if err != nil {
			return &s, fmt.Errorf("Could not add Graphite storage backend: %v\n", err)
		}
	}

	if c.Storage.Dogstatsd.Host != "" {
		err = s.AddEngine(ctx, wg, "dogstatsd", c)
		if err != nil {
			return &s, fmt.Errorf("Could not add dogstatsd storage backend: %v\n", err)
		}
	}

	return &s, nil
}

// AddEngine adds a new StorageEngine of name engineName to our Storage object
func (s *Storage) AddEngine(ctx context.Context, wg *sync.WaitGroup, engineName string, c *Config) error {
	switch engineName {
	case "graphite":
		se := StorageEngine{}
		se.I = NewGraphiteStorage(c)
		se.C = se.I.StartStorageEngine(ctx, wg)
		s.Engines = append(s.Engines, se)
	case "dogstatsd":
		se := StorageEngine{}
		se.I = NewDogstatsdStorage(c)
		se.C = se.I.StartStorageEngine(ctx, wg)
		s.Engines = append(s.Engines, se)
	}

	return nil
}

// metricDistributor receives metrics from gathers and fans them out to the various
// storage backends
func (s *Storage) metricDistributor(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-s.MetricDistributor:
			for _, e := range s.Engines {
				e.C <- m
			}
		case <-ctx.Done():
			log.Println("Cancellation request received.  Cancelling metric distributor.")
			return nil
		}
	}
}
