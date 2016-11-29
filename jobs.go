package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// Job holds a single Selenium
type Job struct {
	Name     string   `yaml:"name"`
	URL      string   `yaml:"url"`
	Type     string   `yaml:"type"`
	Interval uint16   `yaml:"interval"`
	Cookies  []Cookie `yaml:"cookies,omitempty"`
}

// JobRunner holds channels and state related to running Jobs
type JobRunner struct {
	JobChan chan<- Job
	WG      sync.WaitGroup
}

// NewJobRunner returns as JobRunner
func NewJobRunner() *JobRunner {
	jr := JobRunner{}

	jr.JobChan = make(chan Job, 10)

	return &jr
}

// runJob executes the job on a Ticker interval
func runJob(ctx context.Context, wg *sync.WaitGroup, j Job, jchan chan<- Job, seleniumServer string, storage *Storage) {
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
				go RunSimpleTest(j, storage, ctx)
			default:
				// We run Selenium tests by default
				RunSeleniumTest(j, seleniumServer, storage)
			}
		case <-ctx.Done():
			log.Println("Cancellation request received.  Cancelling job runner.")
			return
		}
	}

}

// StartJobs launches all configured jobs
func StartJobs(ctx context.Context, wg *sync.WaitGroup, c *Config, storage *Storage) {
	jr := NewJobRunner()

	jobs := c.Jobs
	seleniumServer := c.Selenium.URL

	rand.Seed(time.Now().Unix())

	for _, j := range jobs {

		// If we've been provided with an offset for staggering jobs, sleep for a random
		// time interval (where: 0 < sleepDur < offset) before starting that job's timer
		if c.Selenium.JobStaggerOffset > 0 {
			sleepDur := time.Duration(rand.Int31n(c.Selenium.JobStaggerOffset*1000)) * time.Millisecond
			fmt.Println("Sleeping for", sleepDur, "before starting next job")
			time.Sleep(sleepDur)
		}

		log.Println("Launching job -> ", j.URL)
		go runJob(ctx, wg, j, jr.JobChan, seleniumServer, storage)
	}

}
