package main

import (
	"time"
)

// makeMetric creates a Metric for a given timing name and value
func makeMetric(timing string, value float64, name string, url string, tags map[string]string) Metric {

	m := Metric{
		Job:       name,
		URL:       url,
		Timing:    timing,
		Value:     value,
		Timestamp: time.Now(),
		Tags:      tags,
	}

	return m
}
