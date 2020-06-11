package main

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

type Job interface {
	StartJob()
}

type JobConfig interface {
	GetJobName() string
}

type JobManagement struct {
	ctx        context.Context
	wg         sync.WaitGroup
	storage    *Storage
	httpclient *http.Client
}

func NewJob(config JobConfig, m *JobManagement) (Job, error) {
	switch c := config.(type) {
	case *SimpleJobConfig:
		return &SimpleJob{
			config:  c,
			wg:      m.wg,
			ctx:     m.ctx,
			storage: m.storage,
			client:  m.httpclient}, nil
	case *SeleniumJobConfig:
		return &SeleniumJob{
			config:  c,
			wg:      m.wg,
			ctx:     m.ctx,
			storage: m.storage}, nil
	default:
		return nil, errors.New("unknown config type")
	}
}

// type JobManagerRunner struct {
// 	ctx     context.Context
// 	JobChan chan<- Job
// 	WG      sync.WaitGroup
// 	Client  *http.Client
// 	Storage *Storage
// }
