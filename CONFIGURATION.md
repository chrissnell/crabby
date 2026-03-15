# Crabby Configuration Reference
Crabby is configured by means of a YAML file that's passed via the `-config` command line parameter when you start Crabby.

## `general` - General configuration options

| Field Name | Description |
| ---------- | ----------- |
| `request-timeout` | Timeout for HTTP requests (Go duration string, default: `15s`) |
| `user-agent` | Custom User-Agent header sent with all HTTP requests (defaults to `crabby/<version>`) |
| `report-internal-metrics` | Report internal runtime metrics (heap, goroutines) to storage backends. `true` or `false` (default: `false`) |
| `internal-metrics-gathering-interval` | How often to gather internal metrics, in seconds (default: `15`) |
| `tags` | Global tags applied to all jobs and their metrics. Per-job tags override globals on name conflict. |

## `jobs` - Configuring pages and URLs to test
The top-level `jobs` array holds all of the sites and URLs that Crabby will test.  There are three types of probes: `simple`, `browser`, and `api`.

### Common job fields

| Field Name | Description |
| ---------- | ----------- |
| `name` | The name of this job. Used as the metric name in storage backends. |
| `type` | Type of probe: `simple`, `browser`, or `api`. |
| `url` | The URL to probe (not used for `api` type — see `steps`). |
| `interval` | How often Crabby runs this job, in seconds. |
| `tags` | Per-job tags applied only to this job's metrics. |

### `simple` job fields

| Field Name | Description |
| ---------- | ----------- |
| `method` | HTTP method to use (default: `GET`). |
| `header` | Map of HTTP headers to send with the request. |
| `cookies` | List of cookies to send with the request (see below). |

### `browser` job fields

| Field Name | Description |
| ---------- | ----------- |
| `headless` | Run Chrome in headless mode. `true` or `false` (default: inherited from `browser` section). |
| `remote-url` | Override the Chrome DevTools Protocol URL for this job. |
| `cookies` | List of cookies to set before loading the page. |

### `api` job fields

API jobs define a multi-step workflow. Instead of a single `url`, they use a `steps` array.

| Field Name | Description |
| ---------- | ----------- |
| `steps` | Array of sequential HTTP requests (see below). |

#### `steps` - API job steps

Each step runs in order.  Responses from earlier steps can be referenced in later steps using `{{ step_name.field.nested_field }}` template syntax.

| Field Name | Description |
| ---------- | ----------- |
| `name` | Step name (used for template references from later steps). |
| `url` | The URL to request. May contain template references. |
| `method` | HTTP method (`GET`, `POST`, etc.). |
| `header` | Map of HTTP headers. |
| `body` | Request body string. May contain template references. |
| `content-type` | Shorthand for setting the Content-Type header. |
| `timeout` | Per-step timeout (Go duration string). |
| `cookies` | List of cookies to send. |
| `tags` | Per-step tags. |

### `cookies`
The optional `cookies` array holds cookies to be sent with HTTP requests.

| Field Name | Description |
| ---------- | ----------- |
| `name` | Cookie name. |
| `domain` | Domain or subdomain for which the cookie is valid. |
| `path` | Path for which the cookie is valid (typically `/`). |
| `value` | Cookie value. |
| `secure` | `true` or `false`. If `true`, cookie is only sent over HTTPS. |

## `browser` - Chrome browser configuration
Required only if you have `browser`-type jobs.

| Field Name | Description |
| ---------- | ----------- |
| `remote-url` | Chrome DevTools Protocol URL (e.g. `http://localhost:9222`). |
| `headless` | Run Chrome in headless mode. `true` or `false` (default: `true`). |

## `storage` - Metrics handling configuration
The `storage` section holds configuration for metrics and event delivery backends. You can enable any combination of backends simultaneously.

### `prometheus` - Prometheus endpoint

| Field Name | Description |
| ---------- | ----------- |
| `listen-addr` | Address and port for the Prometheus metrics endpoint (e.g. `0.0.0.0:9090`). |
| `metric-namespace` | Prefix for all Prometheus metric names (default: `crabby`). |

### `dogstatsd` - DogStatsD

| Field Name | Description |
| ---------- | ----------- |
| `host` | DogStatsD host (typically `localhost` if running the Datadog Agent locally). |
| `port` | DogStatsD port (typically `8125`). |
| `metric-namespace` | Prefix for all metric names. |

### `influxdb` - InfluxDB v2

| Field Name | Description |
| ---------- | ----------- |
| `host` | InfluxDB HTTP(S) URL (e.g. `http://influxdb:8086`). |
| `token` | InfluxDB API token. |
| `token-file` | Path to a file containing the API token (useful for mounted Kubernetes Secrets). Overrides `token`. |
| `org` | InfluxDB organization. |
| `bucket` | InfluxDB bucket. |
| `metric-namespace` | Prefix for all metric names. |

### `splunk-hec` - Splunk HTTP Event Collector

| Field Name | Description |
| ---------- | ----------- |
| `hec-url` | HEC endpoint URL (e.g. `https://splunk:8088/services/collector`). |
| `token` | HEC authentication token. |
| `token-file` | Path to a file containing the HEC token. Overrides `token`. |
| `source` | Splunk `source` field (default: `crabby`). |
| `host` | Splunk `host` field. |
| `metrics-source-type` | Source type for metric entries. |
| `metrics-index` | Index for metric entries. |
| `events-source-type` | Source type for event entries. |
| `events-index` | Index for event entries. |
| `ca-cert` | Path to a CA certificate for validating the HEC URL. |
| `skip-cert-validation` | Disable TLS certificate validation (testing only). |

### `pagerduty` - PagerDuty V2 Events

| Field Name | Description |
| ---------- | ----------- |
| `routing-key` | PagerDuty integration routing key. See [PagerDuty Services and Integrations](https://support.pagerduty.com/docs/services-and-integrations). |
| `routing-key-file` | Path to a file containing the routing key. Overrides `routing-key`. |
| `event-namespace` | Prefix for event names (default: `crabby`). |
| `client` | Client identifier in PagerDuty events (default: `crabby`). |
| `event-duration` | Minimum duration between duplicate incidents (Go duration string, e.g. `1h`). |

### `log` - Log output

| Field Name | Description |
| ---------- | ----------- |
| `file` | `stdout`, `stderr`, or a file path. |
| `time` | Time formatting options (see below). |
| `format` | Output format options (see below). |

#### `log.time`

| Field Name | Description |
| ---------- | ----------- |
| `format` | [Go time format string](https://golang.org/pkg/time/#Time.Format) (default: `2006/01/02 15:04:05`). |
| `location` | [Go time location](https://golang.org/pkg/time/#LoadLocation) for timestamps (default: `Local`). |

#### `log.format`

| Field Name | Description |
| ---------- | ----------- |
| `metric` | Format string for metric lines (default: `%time [M: %job] %timing: %value (%tags)\n`). |
| `event` | Format string for event lines (default: `%time [E: %name] status: %status (%tags)\n`). |
| `tag` | Format string for individual tags (default: `%name: %value`). |
| `tag-seperator` | String used to join tags. |

##### Metric format variables

| Variable | Description |
| -------- | ----------- |
| `%job` | Job name. |
| `%timing` | Timing metric name. |
| `%value` | Recorded value. |
| `%time` | Timestamp. |
| `%url` | Job URL. |
| `%tags` | Formatted tag string. |

##### Event format variables

| Variable | Description |
| -------- | ----------- |
| `%event` | Event name. |
| `%status` | HTTP status code. |
| `%time` | Timestamp. |
| `%tags` | Formatted tag string. |

##### Tag format variables

| Variable | Description |
| -------- | ----------- |
| `%name` | Tag name. |
| `%value` | Tag value. |
