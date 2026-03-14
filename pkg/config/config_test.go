package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "crabby-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		wantErr   bool
		errSubstr string
		check     func(t *testing.T, c ServiceConfig)
	}{
		{
			name: "valid config with jobs and storage",
			yaml: `
jobs:
  - name: google
    type: selenium
    url: https://www.google.com
    interval: 30
  - name: github
    type: simple
    url: https://github.com
    interval: 60
storage:
  influxdb:
    host: https://influx.example.com:8086
    token: my-token
    org: my-org
    bucket: crabby
  dogstatsd:
    host: localhost
    port: 8125
    metric-namespace: crabby
  prometheus:
    listen-addr: ":9090"
    metric-namespace: crabby
`,
			check: func(t *testing.T, c ServiceConfig) {
				if len(c.Jobs) != 2 {
					t.Errorf("expected 2 jobs, got %d", len(c.Jobs))
				}
				if c.Storage.InfluxDB.Host != "https://influx.example.com:8086" {
					t.Errorf("expected influxdb host, got %q", c.Storage.InfluxDB.Host)
				}
				if c.Storage.InfluxDB.Bucket != "crabby" {
					t.Errorf("expected influxdb bucket crabby, got %q", c.Storage.InfluxDB.Bucket)
				}
				if c.Storage.Dogstatsd.Host != "localhost" {
					t.Errorf("expected dogstatsd host localhost, got %q", c.Storage.Dogstatsd.Host)
				}
				if c.Storage.Prometheus.ListenAddr != ":9090" {
					t.Errorf("expected prometheus listen-addr :9090, got %q", c.Storage.Prometheus.ListenAddr)
				}
			},
		},
		{
			name: "valid config with general settings",
			yaml: `
general:
  request-timeout: 10s
  report-internal-metrics: true
  internal-metrics-gathering-interval: 15
  user-agent: crabby/test
  tags:
    env: production
    region: us-east-1
jobs:
  - name: test
    type: simple
    url: https://example.com
`,
			check: func(t *testing.T, c ServiceConfig) {
				if c.General.RequestTimeout != "10s" {
					t.Errorf("expected request-timeout 10s, got %q", c.General.RequestTimeout)
				}
				if !c.General.ReportInternalMetrics {
					t.Error("expected report-internal-metrics true")
				}
				if c.General.InternalMetricsInterval != 15 {
					t.Errorf("expected internal-metrics-gathering-interval 15, got %d", c.General.InternalMetricsInterval)
				}
				if c.General.UserAgent != "crabby/test" {
					t.Errorf("expected user-agent crabby/test, got %q", c.General.UserAgent)
				}
				if c.General.Tags["env"] != "production" {
					t.Errorf("expected tag env=production, got %q", c.General.Tags["env"])
				}
				if c.General.Tags["region"] != "us-east-1" {
					t.Errorf("expected tag region=us-east-1, got %q", c.General.Tags["region"])
				}
			},
		},
		{
			name:      "no jobs returns error",
			yaml:      "storage:\n  influxdb:\n    host: localhost\n",
			wantErr:   true,
			errSubstr: "no jobs configured",
		},
		{
			name:      "empty jobs list returns error",
			yaml:      "jobs: []\n",
			wantErr:   true,
			errSubstr: "no jobs configured",
		},
		{
			name:      "invalid YAML returns error",
			yaml:      "jobs:\n  - name: test\n\t\tbadindent: true\n",
			wantErr:   true,
			errSubstr: "parsing config",
		},
		{
			name: "removed backends graphite and riemann are silently ignored",
			yaml: `
jobs:
  - name: test
    type: simple
    url: https://example.com
storage:
  graphite:
    host: graphite.example.com
    port: 2003
  riemann:
    host: riemann.example.com
    port: 5555
  influxdb:
    host: https://influx.example.com:8086
    bucket: crabby
`,
			check: func(t *testing.T, c ServiceConfig) {
				if len(c.Jobs) != 1 {
					t.Errorf("expected 1 job, got %d", len(c.Jobs))
				}
				// graphite and riemann should be silently ignored since
				// they are not defined in StorageConfig
				if c.Storage.InfluxDB.Host != "https://influx.example.com:8086" {
					t.Errorf("expected influxdb host, got %q", c.Storage.InfluxDB.Host)
				}
				// Verify no error occurred - the removed backends are just ignored
			},
		},
		{
			name: "valid config with splunk-hec storage",
			yaml: `
jobs:
  - name: test
    type: simple
    url: https://example.com
storage:
  splunk-hec:
    token: my-token
    hec-url: https://splunk.example.com:8088
    host: myhost
    source: crabby
    metrics-source-type: crabby_metrics
    metrics-index: metrics
    events-source-type: crabby_events
    events-index: main
    skip-cert-validation: true
`,
			check: func(t *testing.T, c ServiceConfig) {
				if c.Storage.SplunkHec.Token != "my-token" {
					t.Errorf("expected splunk token my-token, got %q", c.Storage.SplunkHec.Token)
				}
				if c.Storage.SplunkHec.HecURL != "https://splunk.example.com:8088" {
					t.Errorf("expected splunk hec-url, got %q", c.Storage.SplunkHec.HecURL)
				}
				if !c.Storage.SplunkHec.SkipCertificateValidation {
					t.Error("expected skip-cert-validation true")
				}
			},
		},
		{
			name: "valid config with log storage",
			yaml: `
jobs:
  - name: test
    type: simple
    url: https://example.com
storage:
  log:
    file: /var/log/crabby.log
    format:
      metric: "{{.Name}} {{.Value}}"
      event: "{{.Name}} {{.Status}}"
      tag: "{{.Key}}={{.Value}}"
      tag-seperator: ","
    time:
      location: UTC
      format: "2006-01-02T15:04:05Z07:00"
`,
			check: func(t *testing.T, c ServiceConfig) {
				if c.Storage.Log.File != "/var/log/crabby.log" {
					t.Errorf("expected log file /var/log/crabby.log, got %q", c.Storage.Log.File)
				}
				if c.Storage.Log.Time.Location != "UTC" {
					t.Errorf("expected time location UTC, got %q", c.Storage.Log.Time.Location)
				}
				if c.Storage.Log.Format.TagSeparator != "," {
					t.Errorf("expected tag separator comma, got %q", c.Storage.Log.Format.TagSeparator)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.yaml)

			cfg, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "reading config") {
		t.Errorf("expected error containing 'reading config', got %q", err.Error())
	}
}

func TestValidateNoJobs(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServiceConfig
	}{
		{
			name: "nil jobs slice",
			cfg:  ServiceConfig{},
		},
		{
			name: "empty jobs slice",
			cfg:  ServiceConfig{Jobs: []yaml.Node{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "no jobs configured") {
				t.Errorf("expected 'no jobs configured' error, got %q", err.Error())
			}
		})
	}
}
