package main

import (
	"log"
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
func runJob(j Job, jchan chan<- Job, seleniumServer string) {
	jobTicker := time.NewTicker(time.Duration(j.Interval) * time.Second)

	for {
		select {
		case <-jobTicker.C:
			err := RunTest(j, seleniumServer)
			if err != nil {
				log.Println("ERROR EXECUTING JOB:", j.URL)
			}
		}
	}

}

// StartJobs launches all configured jobs
func StartJobs(jobs []Job, seleniumServer string) {
	jr := NewJobRunner()

	for _, j := range jobs {
		log.Println("Launching job -> ", j.URL)
		log.Printf("launch job -- -- --> %#v\n", j)
		go runJob(j, jr.JobChan, seleniumServer)
	}

}
