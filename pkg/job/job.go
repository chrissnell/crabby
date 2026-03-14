package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gopkg.in/yaml.v3"
)

// Runner executes a job and returns collected metrics and events.
type Runner interface {
	Run(ctx context.Context) ([]Metric, []Event, error)
}

// Job represents a schedulable monitoring job.
type Job interface {
	Name() string
	Interval() time.Duration
	Runner
}

// JobOptions holds shared configuration passed to job factories.
type JobOptions struct {
	GlobalTags     map[string]string
	RequestTimeout time.Duration
	UserAgent      string
}

// JobFactory creates jobs from YAML configuration.
type JobFactory interface {
	Type() string
	Create(cfg yaml.Node, opts JobOptions) (Job, error)
}

// MetricEventSender receives metrics and events from job runs.
type MetricEventSender interface {
	SendMetrics(ctx context.Context, metrics []Metric)
	SendEvents(ctx context.Context, events []Event)
}

// JobManager manages job registration and scheduling.
type JobManager struct {
	factories map[string]JobFactory
	jobs      []Job
	sender    MetricEventSender
}

// NewJobManager creates a new JobManager.
func NewJobManager(sender MetricEventSender) *JobManager {
	return &JobManager{
		factories: make(map[string]JobFactory),
		sender:    sender,
	}
}

// RegisterFactory registers a JobFactory for a given job type.
func (jm *JobManager) RegisterFactory(f JobFactory) {
	jm.factories[f.Type()] = f
}

// BuildJobs creates jobs from raw YAML nodes.
func (jm *JobManager) BuildJobs(nodes []yaml.Node, opts JobOptions) error {
	for i, node := range nodes {
		var header struct {
			Type string `yaml:"type"`
		}
		if err := node.Decode(&header); err != nil {
			return fmt.Errorf("decoding job %d type: %w", i, err)
		}
		if header.Type == "" {
			return fmt.Errorf("job %d: type not specified", i)
		}
		f, ok := jm.factories[header.Type]
		if !ok {
			return fmt.Errorf("job %d: unknown type %q", i, header.Type)
		}
		j, err := f.Create(node, opts)
		if err != nil {
			return fmt.Errorf("creating job %d (%s): %w", i, header.Type, err)
		}
		jm.jobs = append(jm.jobs, j)
	}
	return nil
}

// Run starts all jobs and blocks until ctx is cancelled.
func (jm *JobManager) Run(ctx context.Context) {
	for _, j := range jm.jobs {
		go jm.schedule(ctx, j)
	}
}

func (jm *JobManager) schedule(ctx context.Context, j Job) {
	slog.Info("starting job", "name", j.Name(), "interval", j.Interval())
	ticker := time.NewTicker(j.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics, events, err := j.Run(ctx)
			if err != nil {
				slog.Error("job run failed", "job", j.Name(), "error", err)
				continue
			}
			jm.sender.SendMetrics(ctx, metrics)
			jm.sender.SendEvents(ctx, events)
		case <-ctx.Done():
			slog.Info("stopping job", "name", j.Name())
			return
		}
	}
}

// MergeTags merges job-specific tags with global tags. Job tags take precedence.
func MergeTags(jobTags, globalTags map[string]string) map[string]string {
	merged := make(map[string]string)
	if len(jobTags) == 0 && len(globalTags) == 0 {
		return merged
	}
	for k, v := range jobTags {
		merged[k] = v
	}
	for k, v := range globalTags {
		if _, present := merged[k]; !present {
			merged[k] = v
		}
	}
	return merged
}
