package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// LogConfig describes the YAML-provided configuration for a logging
// storage backend
type LogConfig struct {
	File   string       `yaml:"file"`
	Format FormatConfig `yaml:"format"`
	Time   TimeConfig   `yaml:"time"`
}

// FormatConfig holds format configuration
type FormatConfig struct {
	Metric       string `yaml:"metric"`
	Event        string `yaml:"event"`
	Tag          string `yaml:"tag"`
	TagSeparator string `yaml:"tag-seperator"`
}

// TimeConfig holds timestamp configuration
type TimeConfig struct {
	Location string `yaml:"location"`
	Format   string `yaml:"format"`
}

// LogStorage holds the runtime configuration for the log storage driver
type LogStorage struct {
	Stream     *os.File
	Format     FormatConfig
	Location   *time.Location
	TimeFormat string
}

// StartStorageEngine creates a goroutine loop to receive metrics and send
// them off to the log file
func (l LogStorage) StartStorageEngine(ctx context.Context, wg *sync.WaitGroup) (chan<- Metric, chan<- Event) {
	metricChan := make(chan Metric, 10)
	eventChan := make(chan Event, 10)

	// Start processing the metrics we receive
	go l.processMetricsAndEvents(ctx, wg, metricChan, eventChan)

	return metricChan, eventChan
}

func (l LogStorage) processMetricsAndEvents(ctx context.Context, wg *sync.WaitGroup, mchan <-chan Metric, echan <-chan Event) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case m := <-mchan:
			err := l.sendMetric(m)
			if err != nil {
				log.Println(err)
			}
		case e := <-echan:
			err := l.sendEvent(e)
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			log.Println("Cancellation request recieved.  Cancelling metrics processor.")
			l.Stream.Close()
			return
		}
	}
}

// buildTagFormatString builds a string from our map of tags
func (l LogStorage) buildTagFormatString(tags map[string]string) string {

	if len(tags) == 0 {
		return ""
	}

	var sb strings.Builder
	for name, value := range tags {
		replacer := strings.NewReplacer(
			"%name", name,
			"%value", value,
		)
		sb.WriteString(replacer.Replace(l.Format.Tag))
		sb.WriteString(l.Format.TagSeparator)
	}
	return strings.TrimSuffix(sb.String(), l.Format.TagSeparator)
}

// buildMetricFormatString builds a string from our metric
func (l LogStorage) buildMetricFormatString(m Metric) string {
	replacer := strings.NewReplacer(
		"%job", m.Job,
		"%timing", m.Timing,
		"%value", fmt.Sprintf("%.6g", m.Value),
		"%time", m.Timestamp.In(l.Location).Format(l.TimeFormat),
		"%url", m.URL,
		"%tags", l.buildTagFormatString(m.Tags))
	return replacer.Replace(l.Format.Metric)
}

// buildEventFormatString builds a string from our event
func (l LogStorage) buildEventFormatString(e Event) string {
	replacer := strings.NewReplacer(
		"%name", e.Name,
		"%status", fmt.Sprint(e.ServerStatus),
		"%time", e.Timestamp.In(l.Location).Format(l.TimeFormat),
		"%tags", l.buildTagFormatString(e.Tags))
	return replacer.Replace(l.Format.Event)
}

// sendMetric sends a metric value to the log file
func (l LogStorage) sendMetric(m Metric) error {
	_, err := l.Stream.WriteString(l.buildMetricFormatString(m))
	if err != nil {
		return err
	}
	return nil
}

// sendEvent sends an event to the log file
func (l LogStorage) sendEvent(e Event) error {
	_, err := l.Stream.WriteString(l.buildEventFormatString(e))
	if err != nil {
		return err
	}
	return nil
}

// NewLogStorage creates new log storage object
func NewLogStorage(c ServiceConfig) (LogStorage, error) {
	var outStream *os.File
	var l = LogStorage{}

	switch c.Storage.Log.File {
	case "stdout":
		outStream = os.Stdout
	case "stderr":
		outStream = os.Stderr
	default:
		var err error
		outStream, err = os.OpenFile(c.Storage.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		// Don't defer fileStream close until processor is cancelled.
		if err != nil {
			return l, err
		}
	}

	if c.Storage.Log.Time.Location == "" {
		c.Storage.Log.Time.Location = "Local"
	}

	location, err := time.LoadLocation(c.Storage.Log.Time.Location)
	if err != nil {
		return l, err
	}

	l.TimeFormat = c.Storage.Log.Time.Format
	l.Location = location
	l.Format = c.Storage.Log.Format

	if l.TimeFormat == "" {
		l.TimeFormat = "2006/01/02 15:04:05"
	}
	if l.Format.Metric == "" {
		l.Format.Metric = "%time [M: %job] %timing: %value (%tags)\n"
	}
	if l.Format.Event == "" {
		l.Format.Event = "%time [E: %name] status: %status (%tags)\n"
	}
	if l.Format.Tag == "" {
		l.Format.Tag = "%name: %value"
	}
	if l.Format.TagSeparator == "" {
		l.Format.TagSeparator = ", "
	}

	l.Stream = outStream
	return l, nil
}
