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
func runJob(ctx context.Context, wg *sync.WaitGroup, j Job, jchan chan<- Job, seleniumServer string, storage *Storage, timeout uint) {
	jobTicker := time.NewTicker(time.Duration(j.Interval) * time.Second)

	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-jobTicker.C:
			err := RunTest(j, seleniumServer, storage, timeout)
			if err != nil {
				log.Println("ERROR EXECUTING JOB:", j.URL)
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

	seleniumServer := c.Selenium.URL
	jobs := c.Jobs

	rand.Seed(time.Now().Unix())

	for _, j := range jobs {
		sleepDur := time.Duration(rand.Int31n(30000)) * time.Millisecond
		fmt.Println("Sleeping for", sleepDur, "before starting next job")
		time.Sleep(sleepDur)
		log.Println("Launching job -> ", j.URL)
		go runJob(ctx, wg, j, jr.JobChan, seleniumServer, storage, c.Selenium.PageLoadTimeout)
	}

}
