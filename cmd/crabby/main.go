package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chrissnell/crabby/pkg/config"
	"github.com/chrissnell/crabby/pkg/job"
	"github.com/chrissnell/crabby/pkg/storage"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfgFile := flag.String("config", "config.yaml", "Path to config file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return nil
	}

	c, err := config.Load(*cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := c.ResolveSecrets(); err != nil {
		return fmt.Errorf("resolving secrets: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up storage backends
	dist := storage.NewDistributor()
	if err := setupBackends(dist, c); err != nil {
		return fmt.Errorf("setting up backends: %w", err)
	}
	if err := dist.Start(ctx); err != nil {
		return fmt.Errorf("starting backends: %w", err)
	}
	defer dist.Close()

	// Set up HTTP client for jobs
	var requestTimeout time.Duration
	if c.General.RequestTimeout == "" {
		requestTimeout = 15 * time.Second
	} else {
		requestTimeout, err = time.ParseDuration(c.General.RequestTimeout)
		if err != nil {
			return fmt.Errorf("parsing request timeout: %w", err)
		}
	}

	userAgent := c.General.UserAgent
	if userAgent == "" {
		userAgent = "crabby/" + version
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
		Timeout: requestTimeout,
	}

	// Set up job manager
	jm := job.NewJobManager(dist)
	jm.RegisterFactory(&job.SimpleFactory{Client: httpClient})
	jm.RegisterFactory(&job.APIFactory{Client: httpClient})
	jm.RegisterFactory(&job.BrowserFactory{})

	opts := job.JobOptions{
		GlobalTags:     c.General.Tags,
		RequestTimeout: requestTimeout,
		UserAgent:      userAgent,
	}

	if err := jm.BuildJobs(c.Jobs, opts); err != nil {
		return fmt.Errorf("building jobs: %w", err)
	}

	// Start internal metrics if configured
	if c.General.ReportInternalMetrics {
		internalJob := job.NewInternalMetricsJob(c.General.InternalMetricsInterval)
		go func() {
			ticker := time.NewTicker(internalJob.Interval())
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					metrics, _, _ := internalJob.Run(ctx)
					dist.SendMetrics(ctx, metrics)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	jm.Run(ctx)

	// Wait for shutdown signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	slog.Info("shutting down")
	cancel()

	return nil
}

func setupBackends(dist *storage.Distributor, c config.ServiceConfig) error {
	if c.Storage.Dogstatsd.Host != "" {
		b, err := storage.NewDogstatsdBackend(c.Storage.Dogstatsd)
		if err != nil {
			return fmt.Errorf("dogstatsd: %w", err)
		}
		dist.AddBackend(b)
	}

	if c.Storage.Prometheus.ListenAddr != "" {
		dist.AddBackend(storage.NewPrometheusBackend(c.Storage.Prometheus))
	}

	if c.Storage.InfluxDB.Host != "" {
		b, err := storage.NewInfluxDBBackend(c.Storage.InfluxDB)
		if err != nil {
			return fmt.Errorf("influxdb: %w", err)
		}
		dist.AddBackend(b)
	}

	if c.Storage.Log.File != "" {
		b, err := storage.NewLogBackend(c.Storage.Log)
		if err != nil {
			return fmt.Errorf("log: %w", err)
		}
		dist.AddBackend(b)
	}

	if c.Storage.PagerDuty.RoutingKey != "" {
		b, err := storage.NewPagerDutyBackend(c.Storage.PagerDuty)
		if err != nil {
			return fmt.Errorf("pagerduty: %w", err)
		}
		dist.AddBackend(b)
	}

	if c.Storage.SplunkHec.HecURL != "" {
		var timeout time.Duration
		if c.General.RequestTimeout != "" {
			var err error
			timeout, err = time.ParseDuration(c.General.RequestTimeout)
			if err != nil {
				return fmt.Errorf("parsing splunk timeout: %w", err)
			}
		}
		b, err := storage.NewSplunkHECBackend(c.Storage.SplunkHec, timeout)
		if err != nil {
			return fmt.Errorf("splunk_hec: %w", err)
		}
		dist.AddBackend(b)
	}

	return nil
}
