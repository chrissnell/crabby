package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type JobController struct {
	JobConfigurationURL string
	SeleniumURL         string
	Storage             *Storage
}

// SingleJobRunner holds channels and state related to running Jobs
type SingleJobRunner struct {
	Job         *Job
	Client      *http.Client
	Storage     *Storage
	SeleniumURL string
	ctx         context.Context
	wg          *sync.WaitGroup
}

func NewJobController(config *Config, storage *Storage) *JobController {
	return &JobController{
		JobConfigurationURL: config.General.JobConfigurationURL,
		SeleniumURL:         config.Selenium.URL,
		Storage:             storage,
	}
}

// NewSingleJobRunner returns as SingleJobRunner
func NewSingleJobRunner(ctx context.Context, wg *sync.WaitGroup, job *Job, storage *Storage, seleniumURL string, tags map[string]string) (*SingleJobRunner, error) {
	timeout, err := time.ParseDuration(job.Timeout)
	if err != nil {
		return &SingleJobRunner{}, fmt.Errorf("could not parse job timeout %v: ", job.Timeout, err)
	}

	// Merge the global tags with the per-job tags.  Per-job tags take precidence.
	job.Tags = mergeTags(job.Tags, tags)

	sjr := &SingleJobRunner{
		ctx:         ctx,
		wg:          wg,
		Job:         job,
		Client:      newHTTPClientWithTimeout(timeout),
		Storage:     storage,
		SeleniumURL: seleniumURL,
	}

	return sjr, nil
}

// start executes the job on a Ticker interval
func (sjr *SingleJobRunner) start() {
	jobTicker := time.NewTicker(time.Duration(j.Interval) * time.Second)

	sjr.wg.Add(1)
	defer sjr.wg.Done()

	for {
		select {
		case <-jobTicker.C:
			switch sjr.Job.Type {
			case "selenium":
				sjr.RunSeleniumTest()
			case "simple":
				go sjr.RunSimpleTest()
			default:
				// We run simple tests by default
				go sjr.RunSimpleTest()
			}
		case <-sjr.ctx.Done():
			log.Println("Cancellation request received.  Cancelling job runner.")
			return
		}
	}

}

// makeMetric creates a Metric for a given timing name and value
func (j *Job) makeMetric(timing string, value float64) Metric {
	tags := make(map[string]string)

	if len(j.Tags) != 0 {
		tags = j.Tags
	}

	m := Metric{
		Job:       j.Name,
		URL:       j.URL,
		Timing:    timing,
		Value:     value,
		Timestamp: time.Now(),
		Tags:      tags,
	}

	return m
}

// makeEvent creates an Event from a given status code
func (j *Job) makeEvent(status int) Event {
	e := Event{
		Name:         j.Name,
		ServerStatus: status,
		Timestamp:    time.Now(),
		Tags:         j.Tags,
	}

	// If our event had no (nil) tags, initialze the tags map so that
	// we don't panic if tags are added later on.
	if len(e.Tags) == 0 {
		e.Tags = make(map[string]string)
	}

	return e

}

// StartJobController launches a JobController to manage jobs
func (m *JobController) StartJobController(ctx context.Context, wg *sync.WaitGroup, c *Config, storage *Storage) error {

	if c.General.JobConfigurationURL != "" {
		log.Println("Starting jobs using local job configuration...")
		err := m.startAllJobs(ctx, wg, c.Jobs, c.General.Tags)
		if err != nil {
			return fmt.Errorf("unable to start jobs: ", err)
		}
	} else {
		fetchJobConfiguration(ctx, c.General.JobConfigurationURL)
		jobCtx, cancel := context.WithCancel(ctx)

		err := m.startAllJobs(jobCtx, wg, c.Jobs, c.General.Tags)
		if err != nil {
			return fmt.Errorf("unable to start jobs: ", err)
		}

	}

	// rand.Seed(time.Now().Unix())

	// for _, j := range jobs {

	// 	sjr, err := NewSingleJobRunner(j, m.Storage, m.SeleniumURL)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	// If we've been provided with an offset for staggering jobs, sleep for a random
	// 	// time interval (where: 0 < sleepDur < offset) before starting that job's timer
	// 	if c.Selenium.JobStaggerOffset > 0 {
	// 		sleepDur := time.Duration(rand.Int31n(c.Selenium.JobStaggerOffset*1000)) * time.Millisecond
	// 		fmt.Println("Sleeping for", sleepDur, "before starting next job")
	// 		time.Sleep(sleepDur)
	// 	}

	// 	log.Println("Launching job -> ", j.URL)
	// 	go sjr.start()
	// }

	return nil
}

func (m *JobController) startAllJobs(ctx context.Context, wg *sync.WaitGroup, jobs []Job, globalTags map[string]string) error {
	for _, job := range jobs {
		sjr, err := NewSingleJobRunner(ctx, wg, &job, m.Storage, m.SeleniumURL, globalTags)
		if err != nil {
			return err
		}

		log.Println("Launching job", job.URL)
		go sjr.start()
	}
}

func mergeTags(jobTags map[string]string, globalTags map[string]string) map[string]string {
	mergedTags := make(map[string]string)

	// If we don't have any global tags or job tags, just return an empty map
	if len(jobTags) == 0 && len(globalTags) == 0 {
		return mergedTags
	}

	for k, v := range jobTags {
		mergedTags[k] = v
	}

	for k, v := range globalTags {
		// Add the global tags to the merged tags, but only if they weren't overriden by a job tag
		_, present := mergedTags[k]
		if !present {
			mergedTags[k] = v
		}
	}

	return mergedTags
}

func newHTTPClientWithTimeout(timeout time.Duration) *http.Client {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   timeoutDuration,
		ExpectContinueTimeout: 1 * time.Second,
		// We have to disable keep-alives to keep our server connection time
		// measurements accurate
		DisableKeepAlives: true,
	}

	// Because we allow per-job timeout settings, we need to create http.Clients for each job
	client := &http.Client{
		Transport: tr,
		Timeout:   timeoutDuration,
	}

	return client
}
