package job

import "time"

// Metric holds one metric data point.
type Metric struct {
	Job       string
	URL       string
	Timing    string
	Value     float64
	Timestamp time.Time
	Tags      map[string]string
}

// Event holds one monitoring event.
type Event struct {
	Name         string
	ServerStatus int
	Timestamp    time.Time
	Tags         map[string]string
}

// MakeMetric creates a Metric for a given timing name and value.
func MakeMetric(timing string, value float64, name, url string, tags map[string]string) Metric {
	return Metric{
		Job:       name,
		URL:       url,
		Timing:    timing,
		Value:     value,
		Timestamp: time.Now(),
		Tags:      tags,
	}
}

// MakeEvent creates an Event from a given status code.
func MakeEvent(name string, status int, tags map[string]string) Event {
	e := Event{
		Name:         name,
		ServerStatus: status,
		Timestamp:    time.Now(),
		Tags:         tags,
	}
	if len(e.Tags) == 0 {
		e.Tags = make(map[string]string)
	}
	return e
}
