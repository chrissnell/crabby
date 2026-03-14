/*
Portions of this file were derived from Dave Cheney's httpstat:
https://github.com/davecheney/httpstat

His code is licensed under the MIT License.
Copyright (c) 2016 Dave Cheney
*/

package job

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"github.com/chrissnell/crabby/pkg/cookie"
	"gopkg.in/yaml.v3"
)

// SimpleJobConfig holds the configuration for a simple job.
type SimpleJobConfig struct {
	Name     string            `yaml:"name"`
	URL      string            `yaml:"url"`
	Method   string            `yaml:"method"`
	Interval uint16            `yaml:"interval"`
	Cookies  []cookie.Cookie   `yaml:"cookies,omitempty"`
	Header   map[string]string `yaml:"header,omitempty"`
	Tags     map[string]string `yaml:"tags,omitempty"`
}

// SimpleJob performs a single HTTP request and collects timing metrics.
type SimpleJob struct {
	config    SimpleJobConfig
	client    *http.Client
	tags      map[string]string
	userAgent string
}

func (j *SimpleJob) Name() string            { return j.config.Name }
func (j *SimpleJob) Interval() time.Duration { return time.Duration(j.config.Interval) * time.Second }

// Run executes the HTTP request and returns timing metrics.
func (j *SimpleJob) Run(ctx context.Context) ([]Metric, []Event, error) {
	method := strings.ToUpper(j.config.Method)
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, j.config.URL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}

	for key, value := range j.config.Header {
		req.Header.Add(key, value)
	}
	if len(j.config.Cookies) > 0 {
		req.Header.Add("Cookie", cookie.HeaderString(j.config.Cookies))
	}
	if j.userAgent != "" {
		req.Header.Set("User-Agent", j.userAgent)
	}

	var t0, t1, t2, t3, t4 time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { t0 = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { t1 = time.Now() },
		ConnectStart: func(_, _ string) {
			if t1.IsZero() {
				t1 = time.Now()
			}
		},
		ConnectDone: func(_, addr string, err error) {
			if err != nil {
				return
			}
			t2 = time.Now()
		},
		GotConn:              func(_ httptrace.GotConnInfo) { t3 = time.Now() },
		GotFirstResponseByte: func() { t4 = time.Now() },
	}

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("executing request: %w", err)
	}
	resp.Body.Close()

	t5 := time.Now()
	if t0.IsZero() {
		t0 = t1
	}

	events := []Event{MakeEvent(j.config.Name, resp.StatusCode, j.tags)}

	u, err := url.Parse(j.config.URL)
	if err != nil {
		return nil, events, fmt.Errorf("parsing URL: %w", err)
	}

	mk := func(timing string, value float64) Metric {
		return MakeMetric(timing, value, j.config.Name, j.config.URL, j.tags)
	}

	var metrics []Metric
	switch u.Scheme {
	case "https":
		metrics = []Metric{
			mk("dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000),
			mk("server_connection_duration_milliseconds", t2.Sub(t1).Seconds()*1000),
			mk("tls_handshake_duration_milliseconds", t3.Sub(t2).Seconds()*1000),
			mk("server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000),
			mk("server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000),
			mk("time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000),
		}
	case "http":
		metrics = []Metric{
			mk("dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000),
			mk("server_connection_duration_milliseconds", t3.Sub(t1).Seconds()*1000),
			mk("server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000),
			mk("server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000),
			mk("time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000),
		}
	}

	return metrics, events, nil
}

// SimpleFactory creates SimpleJob instances.
type SimpleFactory struct {
	Client *http.Client
}

func (f *SimpleFactory) Type() string { return "simple" }

func (f *SimpleFactory) Create(cfg yaml.Node, opts JobOptions) (Job, error) {
	var c SimpleJobConfig
	if err := cfg.Decode(&c); err != nil {
		return nil, fmt.Errorf("decoding simple job config: %w", err)
	}
	return &SimpleJob{
		config:    c,
		client:    f.Client,
		tags:      MergeTags(c.Tags, opts.GlobalTags),
		userAgent: opts.UserAgent,
	}, nil
}
