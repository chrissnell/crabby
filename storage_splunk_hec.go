package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// SplunkHecConfig describes the YAML-provided configuration for a Splunk HEC
// storage backend.
type SplunkHecConfig struct {
	Token                     string `yaml:"token"`
	Tenant                    string `yaml:"tenant"`
	Port                      int    `yaml:"port"`
	Host                      string `yaml:"host"`
	Source                    string `yaml:"source"`
	MetricsSourceType         string `yaml:"metrics-source-type"`
	MetricsIndex              string `yaml:"metrics-index"`
	EventsSourceType          string `yaml:"events-source-type"`
	EventsIndex               string `yaml:"events-index"`
	SkipCertificateValidation bool   `yaml:"skip-cert-validation"`
}

// SplunkHecStorage holds the configuration of a Splunk HEC storage backend
type SplunkHecStorage struct {
	client *http.Client
	config SplunkHecConfig
	url    string
}

// NewSplunkHecStorage sets up a new Splunk HEC storage backend
func NewSplunkHecStorage(c *Config) (SplunkHecStorage, error) {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if c.Storage.SplunkHec.SkipCertificateValidation {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	var requestTimeout time.Duration
	var err error

	if c.General.RequestTimeout == "" {
		requestTimeout = 15 * time.Second
	} else {
		requestTimeout, err = time.ParseDuration(c.General.RequestTimeout)
		if err != nil {
			log.Fatalln("could not parse request timeout duration in config:", err)
		}
	}

	httpClient := &http.Client{
		Transport: tr,
		Timeout:   requestTimeout,
	}

	s := SplunkHecStorage{
		client: httpClient,
		config: c.Storage.SplunkHec,
		url:    fmt.Sprintf("https://%s:%d/services/collector", c.Storage.SplunkHec.Tenant, c.Storage.SplunkHec.Port),
	}
	return s, nil
}

// StartStorageEngine creates a go routine to process events and metrics and sned them
// to a Splunk HEC service
func (s SplunkHecStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	eventChan := make(chan Event, 10)
	metricsChan := make(chan Metric, 10)
	go s.processMetricsAndEvents(ctx, wg, metricsChan, eventChan)
	return metricsChan, eventChan
}

func (s SplunkHecStorage) processMetricsAndEvents(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric, echan <-chan Event) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case m, ok := <-mchan:
			if !ok {
				log.Println("Event channel closed. Cancelling metrics and events process.")
				return
			}
			err := s.sendMetric(m)
			if err != nil {
				log.Println(err)
			}
		case e, ok := <-echan:
			if !ok {
				log.Println("Event channel closed. Cancelling metrics and events process.")
				return
			}
			err := s.sendEvent(e)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request recieved. Cancelling metrics processop.")
			return
		}
	}
}

func (s SplunkHecStorage) sendMetric(m Metric) error {
	sourceType := "metric"
	index := "main"

	if s.config.MetricsSourceType != "" {
		sourceType = s.config.MetricsSourceType
	}

	if s.config.MetricsIndex != "" {
		index = s.config.MetricsIndex
	}
	return s.sendMetricOrEvent(index, sourceType, m.Timestamp, m)
}

func (s SplunkHecStorage) sendEvent(e Event) error {
	sourceType := "event"
	index := "main"

	if s.config.EventsSourceType != "" {
		sourceType = s.config.EventsSourceType
	}

	if s.config.EventsIndex != "" {
		index = s.config.EventsIndex
	}
	return s.sendMetricOrEvent(index, sourceType, e.Timestamp, e)
}

func (s SplunkHecStorage) sendMetricOrEvent(index, sourceType string, ts time.Time, m interface{}) error {
	payload, err := json.Marshal(hecEvent{
		Time:       ts.UnixNano() / 1e6,
		Host:       s.config.Host,
		Source:     s.config.Source,
		SourceType: sourceType,
		Index:      index,
		Event:      m,
	})

	if err != nil {
		return err
	}

	return s.sendHecEvent(payload)
}

func (s SplunkHecStorage) sendHecEvent(event []byte) error {
	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewBuffer(event))

	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Splunk "+s.config.Token)

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		err = fmt.Errorf("unable to send event through Splunc HEC, response: %+v", res)
	}
	return err
}

type hecEvent struct {
	Time       int64       `json:"time"`
	Host       string      `json:"host"`
	Source     string      `json:"source"`
	SourceType string      `json:"sourcetype"`
	Index      string      `json:"index"`
	Event      interface{} `json:"event"`
}
