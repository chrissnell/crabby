# Developing on Crabby

## Project Layout
```
cmd/crabby/         Entry point — config loading, backend wiring, job scheduling
pkg/config/         Configuration parsing, validation, and secret file resolution
pkg/job/            Job types and the job manager
  job.go            Job/JobFactory/JobManager interfaces and scheduler
  simple.go         Simple HTTP probe (net/http with httptrace)
  browser.go        Browser probe (chromedp / Chrome DevTools Protocol)
  api.go            Multi-step API probe with response templating
  internal.go       Internal runtime metrics (heap, goroutines)
pkg/storage/        Storage backend implementations
  storage.go        Backend/MetricSender/EventSender interfaces and Distributor
  prometheus.go     Prometheus endpoint
  dogstatsd.go      DogStatsD (Datadog Agent)
  influxdb.go       InfluxDB v2
  splunk_hec.go     Splunk HTTP Event Collector
  pagerduty.go      PagerDuty V2 Events
  log.go            Configurable log output
pkg/cookie/         Cookie handling
helm/crabby/        Helm chart for Kubernetes deployment
example/            Example configuration files
```

## Building
```bash
make build          # Build binary to bin/crabby
make test           # Run tests
make vet            # Run go vet
make docker-build   # Build Docker image
make docker-push    # Push Docker image to ghcr.io
```

## Architecture

### Startup flow (`cmd/crabby/main.go`)
1. Parse config file and resolve secret files (`token-file`, `routing-key-file`)
2. Create a `Distributor` and register enabled storage backends
3. Create a `JobManager`, register job factories (`SimpleFactory`, `BrowserFactory`, `APIFactory`)
4. Build jobs from YAML config nodes — each factory decodes its own config struct
5. Start all backends, then start the job scheduler
6. Block until SIGINT/SIGTERM, then cancel context for clean shutdown

### Job system
Every job implements the `Job` interface:
```go
type Job interface {
    Name() string
    Interval() time.Duration
    Run(ctx context.Context) ([]Metric, []Event, error)
}
```

Jobs are created by `JobFactory` implementations. The `JobManager` reads each job's `type` field from the raw YAML, dispatches to the matching factory, and schedules the resulting job on its own goroutine with a `time.Ticker`.

Each `Run()` call returns metrics (timing measurements) and events (status codes, errors). The `JobManager` sends these to the `Distributor`, which fans them out to all registered backends.

### Storage system
The `Distributor` holds a list of `Backend` instances. On each metric/event delivery, it type-asserts each backend to `MetricSender` or `EventSender` and calls accordingly. Backends that don't implement a given interface are silently skipped — for example, PagerDuty only handles events, not metrics.

## Adding a Job Type

1. Create `pkg/job/your_type.go` with a config struct, a job struct implementing `Job`, and a factory struct implementing `JobFactory`.

2. The factory's `Type()` method returns the string used in the config file's `type` field.

3. Register the factory in `cmd/crabby/main.go`:
   ```go
   jm.RegisterFactory(&job.YourTypeFactory{})
   ```

4. Add tests in `pkg/job/your_type_test.go`.

## Adding a Storage Backend

1. Create `pkg/storage/your_backend.go` implementing `Backend` and whichever of `MetricSender`/`EventSender` apply.

2. Add a config struct (e.g. `YourBackendConfig`) to `StorageConfig` in `pkg/config/config.go` with appropriate `yaml` tags.

3. If your backend uses secrets, add a `token-file` (or similar) field and handle it in `ResolveSecrets()` in `pkg/config/config.go`.

4. Wire it up in `setupBackends()` in `cmd/crabby/main.go` — check for a non-empty sentinel config field and call `dist.AddBackend(...)`.

5. Add tests in `pkg/storage/your_backend_test.go`.

6. Document the config fields in `CONFIGURATION.md`.

## Testing
Tests live alongside source files. Run them with:
```bash
make test
```

For a specific package:
```bash
go test ./pkg/job/...
go test ./pkg/storage/...
```
