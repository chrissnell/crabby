package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/yaml.v2"
)

// Config is the base of our configuration data structure
type Config struct {
	General  GeneralConfig  `yaml:"general"`
	Jobs     []Job          `yaml:"jobs"`
	Selenium SeleniumConfig `yaml:"selenium"`
	Storage  StorageConfig  `yaml:"storage,omitempty"`
}

// GeneralConfig holds general configuration for this Crabby instance
type GeneralConfig struct {
	Tags                    map[string]string `yaml:"tags"`
	JobConfigurationURL     string            `yaml:"job-configuration-url"`
	RequestTimeout          string            `yaml:"request-timeout,omitempty"`
	ReportInternalMetrics   bool              `yaml:"report-internal-metrics,omitempty"`
	InternalMetricsInterval uint              `yaml:"internal-metrics-gathering-interval,omitempty"`
}

// JobConfig holds a list of jobs to be run
type JobConfig struct {
	Jobs []Job `yaml:"jobs"`
}

// Job holds a single job to be run
type Job struct {
	Name     string            `yaml:"name"`
	URL      string            `yaml:"url"`
	Type     string            `yaml:"type"`
	Interval string            `yaml:"interval"`
	Timeout  string            `yaml:"timeout"`
	Cookies  []Cookie          `yaml:"cookies,omitempty"`
	Tags     map[string]string `yaml:"tags,omitempty"`
}

// SeleniumConfig holds the configuration for our Selenium service
type SeleniumConfig struct {
	URL              string `yaml:"url"`
	JobStaggerOffset int32  `yaml:"job-stagger-offset"`
}

// StorageConfig holds the configuration for various storage backends.
// More than one storage backend can be used simultaneously
type StorageConfig struct {
	Graphite   GraphiteConfig   `yaml:"graphite,omitempty"`
	InfluxDB   InfluxDBConfig   `yaml:"influxdb,omitempty"`
	Dogstatsd  DogstatsdConfig  `yaml:"dogstatsd,omitempty"`
	Prometheus PrometheusConfig `yaml:"prometheus,omitempty"`
	Riemann    RiemannConfig    `yaml:"riemann,omitempty"`
}

// NewConfig creates an new config object from the given filename.
func NewConfig(filename *string) (*Config, error) {
	cfgFile, err := ioutil.ReadFile(*filename)
	if err != nil {
		return &Config{}, err
	}
	c := Config{}
	err = yaml.Unmarshal(cfgFile, &c)
	if err != nil {
		return &Config{}, err
	}

	if len(c.Jobs) == 0 {
		log.Fatalln("No jobs were configured!")
	}

	return &c, nil
}

func fetchJobConfiguration(ctx context.Context, url string) (JobConfig, error) {
	var cfg JobConfig

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return JobConfig{}, fmt.Errorf("could not create http request to fetch job configuration: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return JobConfig{}, fmt.Errorf("could not fetch job configuration: %v", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 || resp.StatusCode < 200 {
		return JobConfig{}, fmt.Errorf("job configuration fetch returned status %v", resp.StatusCode)
	}

	err = yaml.NewDecoder(resp.Body).Decode(&cfg)

	return cfg, err
}
