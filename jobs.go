package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
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
	JobChan chan<- Job
	WG      sync.WaitGroup
	Client  *http.Client
	Storage *Storage
}

// NewJobRunner returns as JobRunner
func NewJobRunner() *JobRunner {
	jr := JobRunner{}

	jr.JobChan = make(chan Job, 10)

	return &jr
}

// runJob executes the job on a Ticker interval
func runJob(ctx context.Context, wg *sync.WaitGroup, j Job, jchan chan<- Job, seleniumServer string, storage *Storage, client *http.Client) {
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
				go RunSimpleTest(ctx, j, storage, client)
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
func StartJobs(ctx context.Context, wg *sync.WaitGroup, c *Config, storage *Storage, client *http.Client) {
	var cfg JobConfig
	var jobs []Job
	var err error

	// Create a sub-context of the main context.Context that we can cancel when we want to restart
	// jobs.

	jobsCtx, cancel := context.WithCancel(ctx)

	// If a job configuration URL was provided, attempt to fetch it and populate our jobs
	// from it.
	if c.General.JobConfigurationURL != "" {
		cfg, err = fetchConfiguration(ctx, c, client)
		if err != nil {
			log.Fatalln("Error fetching job configuration:", err)
		}

		if len(cfg.Jobs) == 0 {
			log.Fatalln("Job configuration URL returned no jobs")
		}

		jobs = cfg.Jobs
	} else {
		// No job configuration URL was provided, so we'll run the jobs specified in the local
		// configuration file.
		jobs = c.Jobs
	}

	jr := NewJobRunner()

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
		go runJob(jobsCtx, wg, j, jr.JobChan, seleniumServer, storage, client)
	}

}

func fetchConfiguration(ctx context.Context, c *Config, client *http.Client) (JobConfig, error) {
	var cfg JobConfig
	r, err := client.Get(c.General.JobConfigurationURL)
	if err != nil {
		return JobConfig{}, err
	}

	defer r.Body.Close()

	if r.StatusCode >= 400 || r.StatusCode < 200 {
		return JobConfig{}, fmt.Errorf("job configuration fetch returned status %v", r.StatusCode)
	}

	err = yaml.NewDecoder(r.Body).Decode(&cfg)

	return cfg, err
}
