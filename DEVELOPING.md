# Developing on Crabby
## Adding Storage Backends
1. Crabby uses a Go interface called `StorageEngineInterface` to standardize the way that metrics/events storage backends operate.  To implement a new storage backend, you should implement this interface and its three functions:
- `sendMetric` sends a metric to the storage backend.  If your backend does not support metrics (e.g. PagerDuty), make this a no-op
- `sendEvent` sends an event to the storage backend.  If your backend does not support events (e.g. Graphite), make this a no-op
- `StartStorageEngine` starts up the storage engine and returns a pair of channels that are then used by the storage distributor to send off metrics+events to this storage engine

2. Configuration is specified in a struct within each storage engine.  If you are adding a new storage engine called "foo", your `storage_foo.go` file should contain a struct called `FooConfig` that contains all of the necessary configuration variables.  Be sure to add the necessary YAML hints and be sure to document all of your options in the [configuration docs](CONFIGURATION.md).

3. You will also have to add your config struct to `StorageConfig` in [config.go](config.go) so that it's parsed from the config file properly.

4. Next, you will need to update the `NewStorage()` function in [storage.go](storage.go) to include your storage backend.  The best way to do this is to choose a variable from your config struct that will _always_ be present if your storage driver is enabled.  Check the variable and if it's not empty, call `s.AddEngine()` to add your storage driver.

5. Finally, you will need to update `AddEngine()` in [storage.go](storage.go) to add a `case` for your engine in the `switch` statement.  See the other engines for an example of how to do this.

Crabby sends metrics and events out to every enabled storage backend.  There is a `log` backend which logs all metrics and events to the console and that's handy for seeing what's going out to storage.