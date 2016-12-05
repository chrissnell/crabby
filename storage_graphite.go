package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/marpaia/graphite-golang"
)

// GraphiteConfig describes the YAML-provided configuration for a Graphite
// storage backend
type GraphiteConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Protocol  string `yaml:"protocol,omitempty"`
	Namespace string `yaml:"metric-namespace,omitempty"`
}

// GraphiteStorage holds the configuration for a Graphite storage backend
type GraphiteStorage struct {
	GraphiteConn *graphite.Graphite
	Namespace    string
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to Graphite
func (g GraphiteStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	// We're going to declare eventChan here but not initialize the channel because Graphite
	// storage doesn't support Events, only Metrics.  We should never receive an Event on this
	// channel and if something mistakenly sends one, the program will panic.
	var eventChan chan<- Event

	// We *do* support Metrics, so we'll initialize this as a buffered channel
	metricChan := make(chan Metric, 10)

	// Start processing the metrics we receive
	go g.processMetrics(ctx, wg, metricChan)

	return metricChan, eventChan
}

func (g GraphiteStorage) processMetrics(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-mchan:
			err := g.sendMetric(m)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request recieved.  Cancelling metrics processor.")
			return
		}
	}
}

// sendMetric sends a metric value to Graphite
func (g GraphiteStorage) sendMetric(m Metric) error {
	var metricName string

	valStr := strconv.FormatFloat(m.Value, 'f', 3, 64)

	if g.Namespace == "" {
		metricName = fmt.Sprintf("crabby.%v", m.Name)
	} else {
		metricName = fmt.Sprintf("%v.%v", g.Namespace, m.Name)
	}

	if m.Timestamp.IsZero() {
		err := g.GraphiteConn.SimpleSend(metricName, valStr)
		if err != nil {
			log.Printf("Could not send metric %v: %v\n", metricName, err)
			return err
		}
		return nil
	}

	gm := graphite.Metric{
		Name:      metricName,
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

// sendEvent is necessary to implement the StorageEngine interface.
func (g GraphiteStorage) sendEvent(e Event) error {
	var err error
	return err
}

// NewGraphiteStorage sets up a new Graphite storage backend
func NewGraphiteStorage(c *Config) GraphiteStorage {
	var err error
	g := GraphiteStorage{}

	g.Namespace = c.Storage.Graphite.Namespace

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
