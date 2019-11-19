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
	Name     string            `yaml:"name"`
	URL      string            `yaml:"url"`
	Type     string            `yaml:"type"`
	Interval uint16            `yaml:"interval"`
	Cookies  []Cookie          `yaml:"cookies,omitempty"`
	Tags     map[string]string `yaml:"tags,omitempty"`
}

// JobConfig holds a list of jobs to be run
type JobConfig struct {
	Jobs []Job `yaml:"jobs"`
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

// StartJobs launches all configured jobs
func StartJobs(ctx context.Context, wg *sync.WaitGroup, c *Config, storage *Storage, client *http.Client) {
	var jobs []Job

	jobs = c.Jobs

	jr := NewJobRunner(ctx)

	seleniumServer := c.Selenium.URL

	rand.Seed(time.Now().Unix())

	for _, j := range jobs {

		// Merge the global tags with the per-job tags.  Per-job tags take precidence.
		j.Tags = mergeTags(j.Tags, c.General.Tags)

		// If we've been provided with an offset for staggering jobs, sleep for a random
		// time interval (where: 0 < sleepDur < offset) before starting that job's timer
		if c.Selenium.JobStaggerOffset > 0 {
			sleepDur := time.Duration(rand.Int31n(c.Selenium.JobStaggerOffset*1000)) * time.Millisecond
			fmt.Println("Sleeping for", sleepDur, "before starting next job")
			time.Sleep(sleepDur)
		}

		log.Println("Launching job -> ", j.URL)
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
