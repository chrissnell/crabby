package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/chrissnell/crabby/pkg/cookie"
	"gopkg.in/yaml.v3"
)

// BrowserJobConfig holds configuration for a browser job.
type BrowserJobConfig struct {
	Name      string            `yaml:"name"`
	URL       string            `yaml:"url"`
	Interval  uint16            `yaml:"interval"`
	Cookies   []cookie.Cookie   `yaml:"cookies,omitempty"`
	Tags      map[string]string `yaml:"tags,omitempty"`
	RemoteURL string            `yaml:"remote-url,omitempty"`
	Headless  *bool             `yaml:"headless,omitempty"`
}

// BrowserJob performs a browser-based page load and collects timing metrics via chromedp.
type BrowserJob struct {
	config BrowserJobConfig
	tags   map[string]string
}

func (j *BrowserJob) Name() string            { return j.config.Name }
func (j *BrowserJob) Interval() time.Duration { return time.Duration(j.config.Interval) * time.Second }

// performanceTiming mirrors the fields we need from window.performance.timing.
type performanceTiming struct {
	DomainLookupStart float64 `json:"domainLookupStart"`
	DomainLookupEnd   float64 `json:"domainLookupEnd"`
	ConnectStart      float64 `json:"connectStart"`
	ConnectEnd        float64 `json:"connectEnd"`
	RequestStart      float64 `json:"requestStart"`
	ResponseStart     float64 `json:"responseStart"`
	ResponseEnd       float64 `json:"responseEnd"`
	DomLoading        float64 `json:"domLoading"`
	DomComplete       float64 `json:"domComplete"`
}

// Run navigates to the configured URL, extracts performance timings, and returns metrics.
func (j *BrowserJob) Run(ctx context.Context) ([]Metric, []Event, error) {
	allocCtx, allocCancel := j.newAllocator(ctx)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(func(format string, args ...interface{}) {
			slog.Debug(fmt.Sprintf(format, args...), "job", j.config.Name)
		}),
	)
	defer taskCancel()

	// Set a timeout for the entire browser operation
	taskCtx, timeoutCancel := context.WithTimeout(taskCtx, 60*time.Second)
	defer timeoutCancel()

	var actions []chromedp.Action

	// Set cookies before navigation if configured
	if len(j.config.Cookies) > 0 {
		for _, c := range j.config.Cookies {
			cp := cookie.ToCookieParam(c)
			actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
				return network.SetCookie(cp.Name, cp.Value).
					WithDomain(cp.Domain).
					WithPath(cp.Path).
					WithSecure(cp.Secure).
					Do(ctx)
			}))
		}
	}

	// Navigate and wait for page load
	actions = append(actions,
		chromedp.Navigate(j.config.URL),
		chromedp.WaitReady("body"),
	)

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return nil, nil, fmt.Errorf("browser navigation: %w", err)
	}

	// Extract performance timing
	var pt performanceTiming
	err := chromedp.Run(taskCtx, chromedp.Evaluate(`(() => {
		const t = window.performance.timing;
		return {
			domainLookupStart: t.domainLookupStart,
			domainLookupEnd: t.domainLookupEnd,
			connectStart: t.connectStart,
			connectEnd: t.connectEnd,
			requestStart: t.requestStart,
			responseStart: t.responseStart,
			responseEnd: t.responseEnd,
			domLoading: t.domLoading,
			domComplete: t.domComplete
		};
	})()`, &pt))
	if err != nil {
		return nil, nil, fmt.Errorf("extracting performance timing: %w", err)
	}

	mk := func(timing string, value float64) Metric {
		return MakeMetric(timing, value, j.config.Name, j.config.URL, j.tags)
	}

	metrics := []Metric{
		mk("dns_duration_milliseconds", pt.DomainLookupEnd-pt.DomainLookupStart),
		mk("server_connection_duration_milliseconds", pt.ConnectEnd-pt.ConnectStart),
		mk("server_processing_duration_milliseconds", pt.ResponseStart-pt.RequestStart),
		mk("server_response_duration_milliseconds", pt.ResponseEnd-pt.ResponseStart),
		mk("dom_rendering_duration_milliseconds", pt.DomComplete-pt.DomLoading),
		mk("time_to_first_byte_milliseconds", pt.ResponseStart-pt.DomainLookupStart),
	}

	// Use 200 as status since browser loaded the page successfully
	events := []Event{MakeEvent(j.config.Name, 200, j.tags)}

	return metrics, events, nil
}

func (j *BrowserJob) newAllocator(parent context.Context) (context.Context, context.CancelFunc) {
	if j.config.RemoteURL != "" {
		return chromedp.NewRemoteAllocator(parent, j.config.RemoteURL)
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", *j.config.Headless),
	)
	return chromedp.NewExecAllocator(parent, opts...)
}

// BrowserFactory creates BrowserJob instances.
type BrowserFactory struct{}

func (f *BrowserFactory) Type() string { return "browser" }

func (f *BrowserFactory) Create(cfg yaml.Node, opts JobOptions) (Job, error) {
	var c BrowserJobConfig
	if err := cfg.Decode(&c); err != nil {
		return nil, fmt.Errorf("decoding browser job config: %w", err)
	}
	if c.URL == "" {
		return nil, fmt.Errorf("browser job %q: url is required", c.Name)
	}
	if c.Headless == nil {
		t := true
		c.Headless = &t
	}
	return &BrowserJob{
		config: c,
		tags:   MergeTags(c.Tags, opts.GlobalTags),
	}, nil
}
