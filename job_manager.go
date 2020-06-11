package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// A Job is an interface to a single instance of a gatherer
type Job interface {
	StartJob()
}

// JobConfig is an interface to provide a generic configuration for a Job
type JobConfig interface {
	GetJobName() string
}

// JobManager holds various things needed to run and manage all jobs
type JobManager struct {
	ctx           context.Context
	wg            *sync.WaitGroup
	storage       *Storage
	httpclient    *http.Client
	jobs          []interface{}
	globalTags    map[string]string
	serviceConfig ServiceConfig
}

// NewJobManager creates a populated JobManager object with configured http.Client
func NewJobManager(ctx context.Context, wg *sync.WaitGroup, s *Storage, serviceConfig ServiceConfig) (*JobManager, error) {
	var err error

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// We have to disable keep-alives to keep our server connection time
		// measurements accurate
		DisableKeepAlives: true,
	}

	var requestTimeout time.Duration

	if serviceConfig.General.RequestTimeout == "" {
		requestTimeout = 15 * time.Second
	} else {
		requestTimeout, err = time.ParseDuration(serviceConfig.General.RequestTimeout)
		if err != nil {
			log.Fatalln("could not parse request timeout duration in config:", err)
		}
	}

	if serviceConfig.General.UserAgent == "" {
		serviceConfig.General.UserAgent = "crabby/1.0"
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   requestTimeout,
	}

	defer tr.CloseIdleConnections()

	// for _, j := range serviceConfig.Jobs {
	// 	switch j.(type) {
	// 	case *SimpleJobConfig:
	// 		j.(*SimpleJobConfig).Tags = mergeTags(j.(*SimpleJobConfig).Tags, serviceConfig.General.Tags)

	// 		jobs = append(jobs, SimpleJob{
	// 			ctx:     ctx,
	// 			wg:      wg,
	// 			storage: s,
	// 			client:  client,
	// 			config:  j.(SimpleJobConfig),
	// 		})
	// 	case *SeleniumJobConfig:
	// 		j.(*SeleniumJobConfig).Tags = mergeTags(j.(*SeleniumJobConfig).Tags, serviceConfig.General.Tags)

	// 		jobs = append(jobs, SeleniumJob{
	// 			ctx:     ctx,
	// 			wg:      wg,
	// 			storage: s,
	// 			config:  j.(SeleniumJobConfig),
	// 		})
	// 	}
	// }

	return &JobManager{
		ctx:           ctx,
		wg:            wg,
		storage:       s,
		httpclient:    client,
		jobs:          serviceConfig.Jobs,
		globalTags:    serviceConfig.General.Tags,
		serviceConfig: serviceConfig,
	}, nil
}

// Run attempts to build and launch all configured jobs
func (jm *JobManager) Run() error {
	err := jm.BuildJobs()
	if err != nil {
		return fmt.Errorf("Unable to build jobs for JobManager: %v", err)
	}

	log.Println("Starting jobs...")
	jm.StartJobs()

	return nil
}

// BuildJobs assembles properly-configured jobs for the JobManager
func (jm *JobManager) BuildJobs() error {
	for _, j := range jm.serviceConfig.Jobs {
		switch j.(type) {
		case *SimpleJobConfig:
			jm.jobs = append(jm.jobs, jm.newJob(j.(*SimpleJobConfig)))
		case *SeleniumJobConfig:
			jm.jobs = append(jm.jobs, jm.newJob(j.(*SeleniumJobConfig)))
		default:
			return errors.New("unknown config type")
		}
	}
	return nil
}

// StartJobs starts all active jobs
func (jm *JobManager) StartJobs() {
	for _, j := range jm.jobs {
		switch j.(type) {
		case *SimpleJob:
			go j.(*SimpleJob).StartJob()
		case *SeleniumJob:
			go j.(*SeleniumJob).StartJob()
		}
	}
}

// newJob creates a new Job of the appropriate type for the chosen gatherer
func (jm *JobManager) newJob(jobconfig JobConfig) Job {
	switch c := jobconfig.(type) {
	case *SimpleJobConfig:
		jobconfig.(*SimpleJobConfig).Tags = mergeTags(jobconfig.(*SimpleJobConfig).Tags, jm.serviceConfig.General.Tags)
		return &SimpleJob{
			config:  *c,
			wg:      jm.wg,
			ctx:     jm.ctx,
			storage: jm.storage,
			client:  jm.httpclient}
	case *SeleniumJobConfig:
		c.seleniumServer = jm.serviceConfig.Selenium.URL
		jobconfig.(*SeleniumJobConfig).Tags = mergeTags(jobconfig.(*SeleniumJobConfig).Tags, jm.serviceConfig.General.Tags)
		return &SeleniumJob{
			config:  *c,
			wg:      jm.wg,
			ctx:     jm.ctx,
			storage: jm.storage}
	default:
		return &NoOpJob{}
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

// NoOpJobConfig defines a job configuration that does nothing.  Used to detect invalid job types.
type NoOpJobConfig struct {
}

// GetJobName does nothing.  Used to detect invalid job types
func (c *NoOpJobConfig) GetJobName() string {
	return ""
}

// NoOpJob defines a job that does nothing.  Used to detect invalid job types.
type NoOpJob struct{}

// StartJob does nothing.  Used to detect invalid job types.
func (n *NoOpJob) StartJob() {}
