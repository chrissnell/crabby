package main

import (
	"fmt"
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
	Engines []StorageEngine
}

// StorageEngine is an interface for various metrics storage backends
type StorageEngine interface {
	SendMetric(Metric) error
}

// NewStorage creats a Storage object, populated with all configured
// StorageEngines
func NewStorage(c *Config) (*Storage, error) {
	var err error

	s := Storage{}

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
		s.Engines = append(s.Engines, NewGraphiteStorage(c))

	}

	return nil
}
