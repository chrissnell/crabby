package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Config is the base of our configuration data structure
type Config struct {
	General                 GeneralConfig  `yaml:"general"`
	Jobs                    []Job          `yaml:"jobs"`
	Selenium                SeleniumConfig `yaml:"selenium"`
	Storage                 StorageConfig  `yaml:"storage,omitempty"`
	ReportInternalMetrics   bool           `yaml:"report-internal-metrics,omitempty"`
	InternalMetricsInterval uint           `yaml:"internal-metrics-gathering-interval,omitempty"`
}

// GeneralConfig holds general configuration for this Crabby instance
type GeneralConfig struct {
	Tags            map[string]string `yaml:"tags,omitempty"`
	StationName     string            `yaml:"hostname,omitempty"`
	StationLocation string            `yaml:"location,omitempty"`
	StationProvider string            `yaml:"provider,omitempty"`
	RequestTimeout  string            `yaml:"request-timeout,omitempty"`
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
