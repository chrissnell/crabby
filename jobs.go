package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Job holds a single job to be run
type Job struct {
	Step     JobStep   `yaml:",inline"`
	Type     string    `yaml:"type"`
	Interval uint16    `yaml:"interval"`
	Steps    []JobStep `yaml:"steps,omitempty"` // an api job may consist of multiple jobs (steps)
}

type JobStep struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Cookies []Cookie          `yaml:"cookies,omitempty"`
	Header  map[string]string `yaml:"header,omitempty"`
	// if header contains a different content type this overwrites it.
	ContentType string            `yaml:"content-type,omitempty"`
	Body        string            `yaml:"body,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty"`
}

// JobRunner holds channels and state related to running Jobs
type JobRunner struct {
	ctx     context.Context
	JobChan chan<- Job
	WG      sync.WaitGroup
	Client  *http.Client
	Storage *Storage
}

// NewJobRunner returns as JobRunner
func NewJobRunner(ctx context.Context) *JobRunner {
	jr := JobRunner{
		ctx: ctx,
	}

	jr.JobChan = make(chan Job, 10)

	return &jr
}

// runJob executes the job on a Ticker interval
func (jr *JobRunner) runJob(wg *sync.WaitGroup, j Job, seleniumServer string, storage *Storage, client *http.Client) {
	jobTicker := time.NewTicker(time.Duration(j.Interval) * time.Second)

	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-jobTicker.C:
			switch j.Type {
			case "selenium":
				RunSeleniumTest(j, seleniumServer, storage)
			case "simple":
				go RunSimpleTest(jr.ctx, j, storage, client)
			case "api":
				go RunApiTest(jr.ctx, j, storage, client)

			default:
				// We run Selenium tests by default
				RunSeleniumTest(j, seleniumServer, storage)
			}
		case <-jr.ctx.Done():
			log.Println("Cancellation request received.  Cancelling job runner.")
			return
		}
	}

}

// makeMetric creates a Metric for a given timing name and value
func (j *JobStep) makeMetric(timing string, value float64) Metric {
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
func (j *JobStep) makeEvent(status int) Event {
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

// StartJobs launches all configured jobs
func StartJobs(ctx context.Context, wg *sync.WaitGroup, c *Config, storage *Storage, client *http.Client) {
	var jobs []Job

	jobs = c.Jobs

	jr := NewJobRunner(ctx)

	seleniumServer := c.Selenium.URL

	rand.Seed(time.Now().Unix())

	for _, j := range jobs {

		// Merge the global tags with the per-job tags.  Per-job tags take precedence.
		j.Step.Tags = mergeTags(j.Step.Tags, c.General.Tags)
		for _, step := range j.Steps {
			step.Tags = mergeTags(j.Step.Tags, c.General.Tags)
		}

		// If we've been provided with an offset for staggering jobs, sleep for a random
		// time interval (where: 0 < sleepDur < offset) before starting that job's timer
		if c.Selenium.JobStaggerOffset > 0 {
			sleepDur := time.Duration(rand.Int31n(c.Selenium.JobStaggerOffset*1000)) * time.Millisecond
			fmt.Println("Sleeping for", sleepDur, "before starting next job")
			time.Sleep(sleepDur)
		}

		log.Println("Launching job -> ", j.Step.Name, j.Step.URL)
		go jr.runJob(wg, j, seleniumServer, storage, client)
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
