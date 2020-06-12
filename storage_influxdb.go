package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	client "github.com/influxdata/influxdb1-client/v2"
)

// InfluxDBConfig describes the YAML-provided configuration for a InfluxDB
// storage backend
type InfluxDBConfig struct {
	Scheme    string `yaml:"scheme"`
	Host      string `yaml:"host"`
	Username  string `yaml:"username,omitempty"`
	Password  string `yaml:"password,omitempty"`
	Database  string `yaml:"database"`
	Port      int    `yaml:"port,omitempty"`
	Protocol  string `yaml:"protocol,omitempty"`
	Namespace string `yaml:"metric-namespace,omitempty"`
}

// InfluxDBStorage holds the configuration for a InfluxDB storage backend
type InfluxDBStorage struct {
	Namespace    string
	InfluxDBConn client.Client
	DBName       string
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to Graphite
func (i InfluxDBStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	// We're going to declare eventChan here but not initialize the channel because Graphite
	// storage doesn't support Events, only Metrics.  We should never receive an Event on this
	// channel and if something mistakenly sends one, the program will panic.
	var eventChan chan<- Event

	// We *do* support Metrics, so we'll initialize this as a buffered channel
	metricChan := make(chan Metric, 10)

	// Start processing the metrics we receive
	go i.processMetrics(ctx, wg, metricChan)

	return metricChan, eventChan
}

func (i InfluxDBStorage) processMetrics(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-mchan:
			err := i.sendMetric(m)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request recieved.  Cancelling metrics processor.")
			return
		}
	}
}

// sendMetric sends a metric value to InfluxDB
func (i InfluxDBStorage) sendMetric(m Metric) error {
	var metricName string

	if i.Namespace == "" {
		metricName = fmt.Sprintf("crabby.%v", m.Job)
	} else {
		metricName = fmt.Sprintf("%v.%v", i.Namespace, m.Job)
	}

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  i.DBName,
		Precision: "s",
	})

	// Make a map to store the metric name/value pair
	values := map[string]interface{}{m.Timing: m.Value}

	pt, err := client.NewPoint(metricName, m.Tags, values, m.Timestamp)

	if err != nil {
		return fmt.Errorf("Could not create data point for InfluxDB: %v", err)
	}

	bp.AddPoint(pt)

	// Write the batch
	err = i.InfluxDBConn.Write(bp)
	if err != nil {
		return fmt.Errorf("Could not write data point to InfluxDB: %v", err)

	}

	return nil

}

// sendEvent is necessary to implement the StorageEngine interface.
func (i InfluxDBStorage) sendEvent(e Event) error {
	var err error
	return err
}

// NewInfluxDBStorage sets up a new InfluxDB storage backend
func NewInfluxDBStorage(c ServiceConfig) InfluxDBStorage {
	var err error
	i := InfluxDBStorage{}

	i.DBName = c.Storage.InfluxDB.Database
	i.Namespace = c.Storage.InfluxDB.Namespace

	if c.Storage.InfluxDB.Protocol != "udp" && c.Storage.InfluxDB.Scheme == "" {
		c.Storage.InfluxDB.Scheme = "http"
	}

	switch c.Storage.InfluxDB.Protocol {
	case "http":
		url := fmt.Sprintf("%v://%v:%v", c.Storage.InfluxDB.Scheme, c.Storage.InfluxDB.Host, c.Storage.InfluxDB.Port)

		i.InfluxDBConn, err = client.NewHTTPClient(client.HTTPConfig{
			Addr:     url,
			Username: c.Storage.InfluxDB.Username,
			Password: c.Storage.InfluxDB.Password,
		})

		if err != nil {
			log.Println("Warning: could not create InfluxDB connection!", err)
			return InfluxDBStorage{}
		}
	case "udp":
		u := client.UDPConfig{
			Addr: fmt.Sprintf("%v:%v", c.Storage.InfluxDB.Host, c.Storage.InfluxDB.Port),
		}

		i.InfluxDBConn, err = client.NewUDPClient(u)

		if err != nil {
			log.Println("Warning: could not create InfluxDB connection.", err)
			return InfluxDBStorage{}
		}
	default:
		url := fmt.Sprintf("%v://%v:%v", c.Storage.InfluxDB.Scheme, c.Storage.InfluxDB.Host, c.Storage.InfluxDB.Port)

		i.InfluxDBConn, err = client.NewHTTPClient(client.HTTPConfig{
			Addr:     url,
			Username: c.Storage.InfluxDB.Username,
			Password: c.Storage.InfluxDB.Password,
		})

		if err != nil {
			log.Println("Warning: could not create InfluxDB connection!", err)
			return InfluxDBStorage{}
		}
	}

	return i
}
