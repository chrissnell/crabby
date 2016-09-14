package main

import (
	"log"
	"strconv"

	"github.com/marpaia/graphite-golang"
)

// GraphiteConfig describes the YAML-provided configuration for a Graphite
// storage backend
type GraphiteConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol,omitempty"`
}

// GraphiteStorage holds the configuration for a Graphite storage backend
type GraphiteStorage struct {
	GraphiteConn *graphite.Graphite
}

// SendMetric sends a metric value to Graphtie
func (g GraphiteStorage) SendMetric(m Metric) error {
	valStr := strconv.FormatFloat(m.Value, 'f', 3, 64)

	if m.Timestamp.IsZero() {
		err := g.GraphiteConn.SimpleSend(m.Name, valStr)
		if err != nil {
			log.Printf("Could not send metric %v: %v\n", m.Name, err)
			return err
		}
		return nil
	}

	gm := graphite.Metric{
		Name:      m.Name,
		Value:     valStr,
		Timestamp: m.Timestamp.Unix(),
	}
	err := g.GraphiteConn.SendMetric(gm)
	if err != nil {
		log.Printf("Could not send metric %v: %v\n", m.Name, err)
		return err
	}

	return nil
}

// NewGraphiteStorage sets up a new Graphite storage backend
func NewGraphiteStorage(c *Config) GraphiteStorage {
	var err error
	g := GraphiteStorage{}

	switch c.Storage.Graphite.Protocol {
	case "tcp":
		g.GraphiteConn, err = graphite.NewGraphite(c.Storage.Graphite.Host, c.Storage.Graphite.Port)
		if err != nil {
			log.Println("Warning: could not create Graphite connection.  Using no-op dummy driver instead.", err)
			g.GraphiteConn = graphite.NewGraphiteNop(c.Storage.Graphite.Host, c.Storage.Graphite.Port)
		}
	case "udp":
		g.GraphiteConn, err = graphite.NewGraphiteUDP(c.Storage.Graphite.Host, c.Storage.Graphite.Port)
		if err != nil {
			log.Println("Warning: could not create Graphite connection.  Using no-op dummy driver instead.", err)
			g.GraphiteConn = graphite.NewGraphiteNop(c.Storage.Graphite.Host, c.Storage.Graphite.Port)
		}
	case "nop":
		g.GraphiteConn = graphite.NewGraphiteNop(c.Storage.Graphite.Host, c.Storage.Graphite.Port)
	default:
		g.GraphiteConn, err = graphite.NewGraphite(c.Storage.Graphite.Host, c.Storage.Graphite.Port)
		if err != nil {
			log.Println("Warning: could not create Graphite connection.  Using no-op dummy driver instead.", err)
			g.GraphiteConn = graphite.NewGraphiteNop(c.Storage.Graphite.Host, c.Storage.Graphite.Port)
		}
	}

	return g
}
