package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
)

// PagerDutyBackend sends events to PagerDuty.
type PagerDutyBackend struct {
	config          config.PagerDutyConfig
	eventTimestamps map[string]time.Time
}

// NewPagerDutyBackend creates a new PagerDuty backend.
func NewPagerDutyBackend(cfg config.PagerDutyConfig) (*PagerDutyBackend, error) {
	if cfg.RoutingKey == "" {
		return nil, errors.New("missing PagerDuty routing key")
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "crabby"
	}
	if cfg.Client == "" {
		cfg.Client = "crabby"
	}
	if cfg.EventDuration == 0 {
		cfg.EventDuration = time.Hour
	}

	return &PagerDutyBackend{
		config:          cfg,
		eventTimestamps: make(map[string]time.Time),
	}, nil
}

func (p *PagerDutyBackend) Name() string                  { return "pagerduty" }
func (p *PagerDutyBackend) Start(_ context.Context) error { return nil }
func (p *PagerDutyBackend) Close() error                  { return nil }

// SendEvent sends an event to PagerDuty for error responses.
func (p *PagerDutyBackend) SendEvent(ctx context.Context, e job.Event) error {
	if e.ServerStatus < 400 {
		return nil
	}

	eventKey := fmt.Sprintf("%s-%d", e.Name, e.ServerStatus)
	lastOccurrence := p.eventTimestamps[eventKey]
	if !e.Timestamp.After(lastOccurrence.Add(p.config.EventDuration)) {
		return nil
	}

	p.eventTimestamps[eventKey] = e.Timestamp

	eventName := fmt.Sprintf("%v.%v", p.config.Namespace, e.Name)

	var severity string
	if e.ServerStatus < 500 {
		severity = "error"
	} else {
		severity = "critical"
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
			Severity:  severity,
			Timestamp: e.Timestamp.Format("2006-01-02T15:04:05.000-0700"),
			Details:   e.Tags,
		},
	}

	response, err := pagerduty.ManageEventWithContext(ctx, event)
	if err != nil {
		return fmt.Errorf("sending PagerDuty event: %w", err)
	}
	if response.Status != "success" {
		return fmt.Errorf("PagerDuty event failed, response: %+v", response)
	}
	return nil
}
