/*
Portions of this file were derived from Dave Cheney's httpstat:
https://github.com/davecheney/httpstat

His code is licensed under the MIT License.
Copyright (c) 2016 Dave Cheney
*/

package job

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"github.com/chrissnell/crabby/pkg/cookie"
	"gopkg.in/yaml.v3"
)

// JobStep represents a single request within an API job.
type JobStep struct {
	Name        string            `yaml:"name"`
	URL         string            `yaml:"url"`
	Method      string            `yaml:"method"`
	Timeout     string            `yaml:"timeout,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty"`
	Cookies     []cookie.Cookie   `yaml:"cookies,omitempty"`
	Header      map[string]string `yaml:"header,omitempty"`
	ContentType string            `yaml:"content-type,omitempty"`
	Body        string            `yaml:"body,omitempty"`
}

// StepResult holds the outcome of a single API step execution.
type StepResult struct {
	StepName   string
	StatusCode int
	Duration   time.Duration
	Metrics    []Metric
	Events     []Event
	Response   json.RawMessage
	Error      error
}

// StepRunner executes a sequence of API steps.
type StepRunner interface {
	RunSteps(ctx context.Context, steps []JobStep) ([]StepResult, error)
}

// APIJobConfig holds the configuration for an API job.
type APIJobConfig struct {
	Steps    []JobStep         `yaml:"steps"`
	Interval uint16            `yaml:"interval"`
	Tags     map[string]string `yaml:"tags,omitempty"`
}

// APIJob performs a multi-step API test.
type APIJob struct {
	config   APIJobConfig
	client   *http.Client
	tags     map[string]string
	template TemplateEngine
}

func (j *APIJob) Name() string            { return j.config.Steps[0].Name }
func (j *APIJob) Interval() time.Duration { return time.Duration(j.config.Interval) * time.Second }

// Run executes all API steps and returns collected metrics and events.
func (j *APIJob) Run(ctx context.Context) ([]Metric, []Event, error) {
	results, err := j.RunSteps(ctx, j.config.Steps)
	if err != nil {
		return nil, nil, err
	}

	var allMetrics []Metric
	var allEvents []Event
	for _, r := range results {
		allMetrics = append(allMetrics, r.Metrics...)
		allEvents = append(allEvents, r.Events...)
	}
	return allMetrics, allEvents, nil
}

// RunSteps implements StepRunner.
func (j *APIJob) RunSteps(ctx context.Context, steps []JobStep) ([]StepResult, error) {
	responses := make(StepResponses)
	results := make([]StepResult, 0, len(steps))

	for i := range steps {
		result := j.runStep(ctx, i, responses)
		results = append(results, result)
		if result.Error != nil {
			return results, fmt.Errorf("step %d (%s): %w", i, steps[i].Name, result.Error)
		}
	}
	return results, nil
}

func (j *APIJob) runStep(ctx context.Context, stepNum int, responses StepResponses) StepResult {
	step := j.config.Steps[stepNum]
	result := StepResult{StepName: step.Name}
	start := time.Now()

	// Apply per-step timeout if configured
	if step.Timeout != "" {
		d, err := time.ParseDuration(step.Timeout)
		if err != nil {
			result.Error = fmt.Errorf("parsing step timeout %q: %w", step.Timeout, err)
			result.Duration = time.Since(start)
			return result
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}

	method := strings.ToUpper(step.Method)
	if method == "" {
		method = http.MethodGet
	}

	body, err := j.template.Resolve(step.Body, responses)
	if err != nil {
		result.Error = fmt.Errorf("substituting body variables: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	req, err := http.NewRequestWithContext(ctx, method, step.URL, strings.NewReader(body))
	if err != nil {
		result.Error = fmt.Errorf("creating request: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	if err := j.addHeaders(req, step, responses); err != nil {
		result.Error = fmt.Errorf("processing headers: %w", err)
		result.Duration = time.Since(start)
		return result
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
		result.Error = fmt.Errorf("executing request: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	result.StatusCode = resp.StatusCode
	result.Events = []Event{MakeEvent(step.Name, resp.StatusCode, step.Tags)}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		result.Error = fmt.Errorf("reading response body: %w", err)
		result.Duration = time.Since(start)
		return result
	}
	result.Response = respBody
	responses[step.Name] = respBody

	t5 := time.Now()
	result.Duration = t5.Sub(start)
	if t0.IsZero() {
		t0 = t1
	}

	u, err := url.Parse(step.URL)
	if err != nil {
		result.Error = fmt.Errorf("parsing URL: %w", err)
		return result
	}

	mk := func(timing string, value float64) Metric {
		return MakeMetric(timing, value, step.Name, step.URL, j.tags)
	}

	switch u.Scheme {
	case "https":
		result.Metrics = []Metric{
			mk("dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000),
			mk("server_connection_duration_milliseconds", t2.Sub(t1).Seconds()*1000),
			mk("tls_handshake_duration_milliseconds", t3.Sub(t2).Seconds()*1000),
			mk("server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000),
			mk("server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000),
			mk("time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000),
		}
	case "http":
		result.Metrics = []Metric{
			mk("dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000),
			mk("server_connection_duration_milliseconds", t3.Sub(t1).Seconds()*1000),
			mk("server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000),
			mk("server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000),
			mk("time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000),
		}
	}

	return result
}

func (j *APIJob) addHeaders(req *http.Request, step JobStep, responses StepResponses) error {
	req.Header = http.Header{}
	for key, value := range step.Header {
		req.Header.Add(key, value)
	}
	if step.ContentType != "" {
		req.Header["Content-Type"] = []string{step.ContentType}
	}
	for key := range req.Header {
		for i := range req.Header[key] {
			var err error
			if req.Header[key][i], err = j.template.Resolve(req.Header[key][i], responses); err != nil {
				return err
			}
		}
	}
	return nil
}

// APIFactory creates APIJob instances.
type APIFactory struct {
	Client *http.Client
}

func (f *APIFactory) Type() string { return "api" }

func (f *APIFactory) Create(cfg yaml.Node, opts JobOptions) (Job, error) {
	var c APIJobConfig
	if err := cfg.Decode(&c); err != nil {
		return nil, fmt.Errorf("decoding api job config: %w", err)
	}
	if err := validateStepNames(c.Steps); err != nil {
		return nil, err
	}
	return &APIJob{
		config: c,
		client: f.Client,
		tags:   MergeTags(c.Tags, opts.GlobalTags),
	}, nil
}

// validateStepNames ensures all step names within an API job are unique.
func validateStepNames(steps []JobStep) error {
	seen := make(map[string]int, len(steps))
	for i, s := range steps {
		if s.Name == "" {
			return fmt.Errorf("step %d: name is required", i)
		}
		if prev, ok := seen[s.Name]; ok {
			return fmt.Errorf("step %d: duplicate name %q (first used at step %d)", i, s.Name, prev)
		}
		seen[s.Name] = i
	}
	return nil
}
