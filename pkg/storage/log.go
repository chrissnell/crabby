package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
)

// LogBackend writes metrics and events to a log file or stdout/stderr.
type LogBackend struct {
	stream     *os.File
	format     config.FormatConfig
	location   *time.Location
	timeFormat string
}

// NewLogBackend creates a new log backend.
func NewLogBackend(cfg config.LogConfig) (*LogBackend, error) {
	var stream *os.File
	switch cfg.File {
	case "stdout":
		stream = os.Stdout
	case "stderr":
		stream = os.Stderr
	default:
		var err error
		stream, err = os.OpenFile(cfg.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
	}

	if cfg.Time.Location == "" {
		cfg.Time.Location = "Local"
	}
	location, err := time.LoadLocation(cfg.Time.Location)
	if err != nil {
		return nil, fmt.Errorf("loading time location: %w", err)
	}

	l := &LogBackend{
		stream:     stream,
		format:     cfg.Format,
		location:   location,
		timeFormat: cfg.Time.Format,
	}

	if l.timeFormat == "" {
		l.timeFormat = "2006/01/02 15:04:05"
	}
	if l.format.Metric == "" {
		l.format.Metric = "%time [M: %job] %timing: %value (%tags)\n"
	}
	if l.format.Event == "" {
		l.format.Event = "%time [E: %name] status: %status (%tags)\n"
	}
	if l.format.Tag == "" {
		l.format.Tag = "%name: %value"
	}
	if l.format.TagSeparator == "" {
		l.format.TagSeparator = ", "
	}

	return l, nil
}

func (l *LogBackend) Name() string                  { return "log" }
func (l *LogBackend) Start(_ context.Context) error { return nil }
func (l *LogBackend) Close() error                  { return l.stream.Close() }

// SendMetric writes a metric to the log.
func (l *LogBackend) SendMetric(_ context.Context, m job.Metric) error {
	_, err := l.stream.WriteString(l.BuildMetricString(m))
	return err
}

// SendEvent writes an event to the log.
func (l *LogBackend) SendEvent(_ context.Context, e job.Event) error {
	_, err := l.stream.WriteString(l.BuildEventString(e))
	return err
}

// BuildTagString formats tags according to configured format.
func (l *LogBackend) BuildTagString(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	var sb strings.Builder
	for name, value := range tags {
		replacer := strings.NewReplacer("%name", name, "%value", value)
		sb.WriteString(replacer.Replace(l.format.Tag))
		sb.WriteString(l.format.TagSeparator)
	}
	return strings.TrimSuffix(sb.String(), l.format.TagSeparator)
}

// BuildMetricString formats a metric into a log string.
func (l *LogBackend) BuildMetricString(m job.Metric) string {
	replacer := strings.NewReplacer(
		"%job", m.Job,
		"%timing", m.Timing,
		"%value", fmt.Sprintf("%.6g", m.Value),
		"%time", m.Timestamp.In(l.location).Format(l.timeFormat),
		"%url", m.URL,
		"%tags", l.BuildTagString(m.Tags),
	)
	return replacer.Replace(l.format.Metric)
}

// BuildEventString formats an event into a log string.
func (l *LogBackend) BuildEventString(e job.Event) string {
	replacer := strings.NewReplacer(
		"%name", e.Name,
		"%status", fmt.Sprint(e.ServerStatus),
		"%time", e.Timestamp.In(l.location).Format(l.timeFormat),
		"%tags", l.BuildTagString(e.Tags),
	)
	return replacer.Replace(l.format.Event)
}
