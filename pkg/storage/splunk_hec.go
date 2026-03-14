package storage

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
)

// SplunkHECBackend sends metrics and events to Splunk via HTTP Event Collector.
type SplunkHECBackend struct {
	client *http.Client
	config config.SplunkHecConfig
	ctx    context.Context
}

// NewSplunkHECBackend creates a new Splunk HEC backend.
func NewSplunkHECBackend(cfg config.SplunkHecConfig, requestTimeout time.Duration) (*SplunkHECBackend, error) {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if cfg.CaCert != "" {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("loading system cert pool: %w", err)
		}
		if rootCAs == nil {
			return nil, fmt.Errorf("system certificate pool is nil")
		}
		certs, err := os.ReadFile(cfg.CaCert)
		if err != nil {
			return nil, fmt.Errorf("reading ca-cert from %s: %w", cfg.CaCert, err)
		}
		rootCAs.AppendCertsFromPEM(certs)
		tr.TLSClientConfig = &tls.Config{
			RootCAs: rootCAs,
		}
	} else {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: cfg.SkipCertificateValidation,
		}
	}

	if requestTimeout == 0 {
		requestTimeout = 15 * time.Second
	}

	return &SplunkHECBackend{
		client: &http.Client{Transport: tr, Timeout: requestTimeout},
		config: cfg,
	}, nil
}

func (s *SplunkHECBackend) Name() string { return "splunk_hec" }

func (s *SplunkHECBackend) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *SplunkHECBackend) Close() error { return nil }

// SendMetric sends a metric to Splunk HEC.
func (s *SplunkHECBackend) SendMetric(_ context.Context, m job.Metric) error {
	sourceType := "metric"
	index := "main"
	if s.config.MetricsSourceType != "" {
		sourceType = s.config.MetricsSourceType
	}
	if s.config.MetricsIndex != "" {
		index = s.config.MetricsIndex
	}
	return s.send(index, sourceType, m.Timestamp, m)
}

// SendEvent sends an event to Splunk HEC.
func (s *SplunkHECBackend) SendEvent(_ context.Context, e job.Event) error {
	sourceType := "event"
	index := "main"
	if s.config.EventsSourceType != "" {
		sourceType = s.config.EventsSourceType
	}
	if s.config.EventsIndex != "" {
		index = s.config.EventsIndex
	}
	return s.send(index, sourceType, e.Timestamp, e)
}

func (s *SplunkHECBackend) send(index, sourceType string, ts time.Time, data interface{}) error {
	payload, err := json.Marshal(HECEvent{
		Time:       ts.UnixNano() / 1e6,
		Host:       s.config.Host,
		Source:     s.config.Source,
		SourceType: sourceType,
		Index:      index,
		Event:      data,
	})
	if err != nil {
		return fmt.Errorf("marshaling Splunk HEC event: %w", err)
	}

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.HecURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Splunk "+s.config.Token)

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Splunk HEC request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("Splunk HEC returned status %d", res.StatusCode)
	}
	return nil
}

// HECEvent is the JSON payload for Splunk HEC.
type HECEvent struct {
	Time       int64       `json:"time"`
	Host       string      `json:"host"`
	Source     string      `json:"source"`
	SourceType string      `json:"sourcetype"`
	Index      string      `json:"index"`
	Event      interface{} `json:"event"`
}
