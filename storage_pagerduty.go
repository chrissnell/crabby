package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

// PagerDutyConfig describes the YAML-provided configuration for a PagerDuty
// storage backend
type PagerDutyConfig struct {
	Namespace     string        `yaml:"event-namespace,omitempty"`
	RoutingKey    string        `yaml:"routing-key"`
	Client        string        `yaml:"client"`
	EventDuration time.Duration `yaml:"event-duration,omitempty"`
}

// PagerDutyStorage holds the configuration for a PagerDuty storage backend
type PagerDutyStorage struct {
	config          PagerDutyConfig
	Client          *pagerduty.Client
	eventTimestamps map[string]time.Time
}

// NewPagerDutyStorage sets up a new PagerDuty storage backend
func NewPagerDutyStorage(c *Config) (PagerDutyStorage, error) {
	p := PagerDutyStorage{}

	p.config = c.Storage.PagerDuty

	if p.config.RoutingKey == "" {
		return p, errors.New("mising PagerDuty routing key")
	}

	if p.config.Namespace == "" {
		p.config.Namespace = "crabby"
	}

	if p.config.Client == "" {
		p.config.Client = "crabby"
	}

	if p.config.EventDuration == 0 {
		p.config.EventDuration = time.Hour
	}

	p.Client = pagerduty.NewClient(p.config.RoutingKey)
	p.eventTimestamps = map[string]time.Time{}
	return p, nil
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to a Prometheus pushgateway
func (p PagerDutyStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	// PagerDuty storage supports both metrics and events, so we'll initialize both channels
	metricChan := make(chan Metric, 10)
	eventChan := make(chan Event, 10)

	// Start processing the metrics we receive
	go p.processMetricsAndEvents(ctx, wg, metricChan, eventChan)

	return metricChan, eventChan
}

func (p PagerDutyStorage) processMetricsAndEvents(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric, echan <-chan Event) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-mchan:
		case e := <-echan:
			err := p.sendEvent(e)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request recieved.  Cancelling metrics processop.")
			return
		}
	}
}

// sendMetric sends a metric value to PagerDuty
func (p PagerDutyStorage) sendMetric(m Metric) error {
	return errors.New("metrics not supported")
}

// sendEvent sends an event to PagerDuty
func (p PagerDutyStorage) sendEvent(e Event) error {
	// Do not create incidents for non-error responses
	if e.ServerStatus >= 400 {
		eventKey := fmt.Sprintf("%s-%d", e.Name, e.ServerStatus)
		lastOccurrence := p.eventTimestamps[eventKey]
		if e.Timestamp.After(lastOccurrence.Add(p.config.EventDuration)) {
			// Ignore events that happen before the next window
			p.eventTimestamps[eventKey] = e.Timestamp

			var eventName string
			var state string

			eventName = fmt.Sprintf("%v.%v", p.config.Namespace, e.Name)

			if e.ServerStatus < 500 {
				state = "error"
			} else {
				state = "critical"
			}

			dedupKey := fmt.Sprintf("%s-%d", eventKey, e.Timestamp.UnixNano())
			event := pagerduty.V2Event{
				Client:     p.config.Client,
				Action:     "trigger",
				DedupKey:   dedupKey,
				RoutingKey: p.config.RoutingKey,
				Payload: &pagerduty.V2Payload{
					Summary:   fmt.Sprintf("%v returned status %v", eventName, e.ServerStatus),
					Source:    p.config.Client,
					Severity:  state,
					Timestamp: e.Timestamp.Format("2006-01-02T15:04:05.000-0700"),
					Details:   e.Tags,
				},
			}
			response, err := pagerduty.ManageEvent(event)
			if err != nil {
				return fmt.Errorf("unable to send event via Pagerduty API: %v", err)
			}
			if response.Status != "success" {
				return fmt.Errorf("unable to send event via Pagerduty API, response:%+v", response)
			}
		}

	}

	return nil
}