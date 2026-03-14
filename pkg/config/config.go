package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// BrowserConfig holds global browser testing configuration.
type BrowserConfig struct {
	RemoteURL string `yaml:"remote-url,omitempty"`
	Headless  bool   `yaml:"headless,omitempty"`
}

// ServiceConfig is the root configuration.
type ServiceConfig struct {
	General  GeneralConfig `yaml:"general"`
	Jobs     []yaml.Node   `yaml:"jobs"`
	Storage  StorageConfig `yaml:"storage,omitempty"`
	Browser  BrowserConfig `yaml:"browser,omitempty"`
	Selenium BrowserConfig `yaml:"selenium,omitempty"`
}

// GeneralConfig holds general configuration for a Crabby instance.
type GeneralConfig struct {
	Tags                    map[string]string `yaml:"tags"`
	RequestTimeout          string            `yaml:"request-timeout,omitempty"`
	ReportInternalMetrics   bool              `yaml:"report-internal-metrics,omitempty"`
	InternalMetricsInterval uint              `yaml:"internal-metrics-gathering-interval,omitempty"`
	UserAgent               string            `yaml:"user-agent,omitempty"`
}

// StorageConfig holds configuration for storage backends.
type StorageConfig struct {
	InfluxDB   InfluxDBConfig   `yaml:"influxdb,omitempty"`
	Dogstatsd  DogstatsdConfig  `yaml:"dogstatsd,omitempty"`
	PagerDuty  PagerDutyConfig  `yaml:"pagerduty,omitempty"`
	Prometheus PrometheusConfig `yaml:"prometheus,omitempty"`
	Log        LogConfig        `yaml:"log,omitempty"`
	SplunkHec  SplunkHecConfig  `yaml:"splunk-hec,omitempty"`
}

// DogstatsdConfig holds Datadog DogStatsD configuration.
type DogstatsdConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Namespace string `yaml:"metric-namespace"`
}

// PrometheusConfig holds Prometheus configuration.
type PrometheusConfig struct {
	ListenAddr string `yaml:"listen-addr"`
	Namespace  string `yaml:"metric-namespace,omitempty"`
}

// InfluxDBConfig holds InfluxDB v2 configuration.
type InfluxDBConfig struct {
	Host      string `yaml:"host"`
	Token     string `yaml:"token"`
	TokenFile string `yaml:"token-file,omitempty"`
	Org       string `yaml:"org"`
	Bucket    string `yaml:"bucket"`
	Namespace string `yaml:"metric-namespace,omitempty"`
}

// LogConfig holds log file configuration.
type LogConfig struct {
	File   string       `yaml:"file"`
	Format FormatConfig `yaml:"format"`
	Time   TimeConfig   `yaml:"time"`
}

// FormatConfig holds log format configuration.
type FormatConfig struct {
	Metric       string `yaml:"metric"`
	Event        string `yaml:"event"`
	Tag          string `yaml:"tag"`
	TagSeparator string `yaml:"tag-seperator"`
}

// TimeConfig holds timestamp configuration.
type TimeConfig struct {
	Location string `yaml:"location"`
	Format   string `yaml:"format"`
}

// PagerDutyConfig holds PagerDuty configuration.
type PagerDutyConfig struct {
	Namespace      string        `yaml:"event-namespace,omitempty"`
	RoutingKey     string        `yaml:"routing-key"`
	RoutingKeyFile string        `yaml:"routing-key-file,omitempty"`
	Client         string        `yaml:"client"`
	EventDuration  time.Duration `yaml:"event-duration,omitempty"`
}

// SplunkHecConfig holds Splunk HEC configuration.
type SplunkHecConfig struct {
	Token                     string `yaml:"token"`
	TokenFile                 string `yaml:"token-file,omitempty"`
	HecURL                    string `yaml:"hec-url"`
	Host                      string `yaml:"host"`
	Source                    string `yaml:"source"`
	MetricsSourceType         string `yaml:"metrics-source-type"`
	MetricsIndex              string `yaml:"metrics-index"`
	EventsSourceType          string `yaml:"events-source-type"`
	EventsIndex               string `yaml:"events-index"`
	SkipCertificateValidation bool   `yaml:"skip-cert-validation"`
	CaCert                    string `yaml:"ca-cert"`
}

// readSecretFile reads a secret from a file path, trimming whitespace.
// Returns the file contents if path is non-empty, otherwise returns fallback.
func readSecretFile(path, fallback string) (string, error) {
	if path == "" {
		return fallback, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading secret file %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ResolveSecrets resolves file-based secret references into their inline fields.
// When both inline and file variants are set, the file takes precedence.
func (c *ServiceConfig) ResolveSecrets() error {
	if c.Storage.InfluxDB.TokenFile != "" {
		token, err := readSecretFile(c.Storage.InfluxDB.TokenFile, c.Storage.InfluxDB.Token)
		if err != nil {
			return err
		}
		c.Storage.InfluxDB.Token = token
	}
	if c.Storage.SplunkHec.TokenFile != "" {
		token, err := readSecretFile(c.Storage.SplunkHec.TokenFile, c.Storage.SplunkHec.Token)
		if err != nil {
			return err
		}
		c.Storage.SplunkHec.Token = token
	}
	if c.Storage.PagerDuty.RoutingKeyFile != "" {
		key, err := readSecretFile(c.Storage.PagerDuty.RoutingKeyFile, c.Storage.PagerDuty.RoutingKey)
		if err != nil {
			return err
		}
		c.Storage.PagerDuty.RoutingKey = key
	}
	return nil
}

// Load reads and parses a config file.
func Load(path string) (ServiceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("reading config: %w", err)
	}
	var c ServiceConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		return ServiceConfig{}, fmt.Errorf("parsing config: %w", err)
	}
	return c, c.validate()
}

func (c *ServiceConfig) validate() error {
	if len(c.Jobs) == 0 {
		return fmt.Errorf("no jobs configured")
	}
	// Support deprecated "selenium" key as alias for "browser"
	if c.Selenium.RemoteURL != "" {
		slog.Warn("config key 'selenium' is deprecated, use 'browser' instead")
		if c.Browser.RemoteURL == "" {
			c.Browser = c.Selenium
		}
		c.Selenium = BrowserConfig{}
	}
	return nil
}
