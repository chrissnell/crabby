# crabby
**crabby** is a website performance tester that measures page load times and reports the measurements to a collection endpoint for processing, monitoring, and viewing.   Crabby currently supports two types of metrics reporting:

* Graphite (metrics, via TCP or UDP)
* Datadog (metrics and service checks, via API)

![Multi-POP Performance Graph in Grafana](https://chrissnell.github.io/crabby/images/crabby-multi-site-grafana.png "Four crabby nodes sending metrics to Graphite+Grafana")

## Two Types of Performance Measuring
TBD

# Using Crabby

To use it, you edit ```config.yaml``` and specify some URLs that you want it to check, optionally providing some pre-provisioned cookies if the site requires them.  Crabby goes and checks the pages on the specified time interval and collects some basic metrics, which it then sends to all of the configured back-end time series databases, storage services, etc., where they can be charted with a tool like [Grafana](http://grafana.org).
