package main

import (
	"fmt"
	"log"
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
	StartStorageEngine() chan<- Metric
}

// NewStorage creats a Storage object, populated with all configured
// StorageEngines
func NewStorage(c *Config) (*Storage, error) {
	var err error

	s := Storage{}

	// Initialize our channel for passing metrics to the MetricDistributor
	s.MetricDistributor = make(chan Metric, 20)

	// Start our metric distributor to distribute received metrics to storage
	// backends
	go s.metricDistributor()

	// Check the configuration file for various supported storage backends
	// and enable them if found
	if c.Storage.Graphite.Host != "" {
		err = s.AddEngine("graphite", c)
		if err != nil {
			return &s, fmt.Errorf("Could not add Graphite storage backend: %v\n", err)
		}
	}

	return &s, nil
}

// AddEngine adds a new StorageEngine of name engineName to our Storage object
func (s *Storage) AddEngine(engineName string, c *Config) error {
	switch engineName {
	case "graphite":
		se := StorageEngine{}
		se.I = NewGraphiteStorage(c)
		se.C = se.I.StartStorageEngine()
		s.Engines = append(s.Engines, se)
	}

	return nil
}

// metricDistributor receives metrics from gathers and fans them out to the various
// storage backends
func (s *Storage) metricDistributor() error {
	for {
		select {
		case m := <-s.MetricDistributor:
			for _, e := range s.Engines {
				log.Println("Metric Distributor :: Sending metric", m.Name)
				e.C <- m
			}
		}
	}
}
