package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// ServiceConfig is the base of our configuration data structure
type ServiceConfig struct {
	General  GeneralConfig   `yaml:"general"`
	Jobs     []MetaJobConfig `yaml:"jobs"`
	Selenium SeleniumConfig  `yaml:"selenium"`
	Storage  StorageConfig   `yaml:"storage,omitempty"`
}

// GeneralConfig holds general configuration for this Crabby instance
type GeneralConfig struct {
	Tags                    map[string]string `yaml:"tags"`
	JobConfigurationURL     string            `yaml:"job-configuration-url,omitempty"`
	RequestTimeout          string            `yaml:"request-timeout,omitempty"`
	ReportInternalMetrics   bool              `yaml:"report-internal-metrics,omitempty"`
	InternalMetricsInterval uint              `yaml:"internal-metrics-gathering-interval,omitempty"`
	UserAgent               string            `yaml:"user-agent,omitempty"`
}

// MetaJobConfig is an intermediate holding place for Job configurations.  It has members
// to account for every possible configuration member for all of our types of jobs.  If
// you are adding a new type of job with new configuration options, be sure to add those
// options to this struct.
type MetaJobConfig struct {
	Type     string    `yaml:"type"`
	Step     JobStep   `yaml:",inline"`
	Steps    []JobStep `yaml:"steps,omitempty"`
	Interval uint16    `yaml:"interval"`
}

// JobStep represents a single request.  Jobs are usually a single JobStep but some gatherers support
// multiple JobStep's.
type JobStep struct {
	Name        string            `yaml:"name"`
	URL         string            `yaml:"url"`
	Method      string            `yaml:"method"`
	Tags        map[string]string `yaml:"tags,omitempty"`
	Cookies     []Cookie          `yaml:"cookies,omitempty"`
	Header      map[string]string `yaml:"header,omitempty"`
	ContentType string            `yaml:"content-type,omitempty"` // overwrites Content-type header
	Body        string            `yaml:"body,omitempty"`
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
	PagerDuty  PagerDutyConfig  `yaml:"pagerduty,omitempty"`
	Prometheus PrometheusConfig `yaml:"prometheus,omitempty"`
	Riemann    RiemannConfig    `yaml:"riemann,omitempty"`
	Log        LogConfig        `yaml:"log,omitempty"`
	SplunkHec  SplunkHecConfig  `yaml:"splunk-hec,omitempty"`
}

// NewConfig creates an new config object from the given filename.
func NewConfig(filename *string) (ServiceConfig, error) {
	cfgFile, err := ioutil.ReadFile(*filename)
	if err != nil {
		return ServiceConfig{}, err
	}
	c := ServiceConfig{}
	err = yaml.Unmarshal(cfgFile, &c)
	if err != nil {
		return ServiceConfig{}, err
	}

	if len(c.Jobs) == 0 {
		log.Fatalln("No jobs were configured!")
	}

	return c, nil
}
