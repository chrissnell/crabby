# crabby
**crabby** is a website performance tester that measures page load times and reports the measurements to a collection endpoint for processing, monitoring, and viewing.   Crabby can collect and report these metrics:

- DNS resolution time
- TCP connection time
- TLS negotiation time
- HTTP Response Code
- Remote server processing time
- Time to first byte (TTFB)
- Server response time
- DOM rendering time

Crabby currently supports two types of metrics delivery:

* Graphite - Time measurements as metrics using Carbon protocol over TCP or UDP
* Datadog API - Time measurements as metrics; HTTP response codes as service check

![Multi-POP Performance Graph in Grafana](https://chrissnell.github.io/crabby/images/crabby-multi-site-grafana.png "Four crabby nodes sending metrics to Graphite for display in Grafana")

## Two Types of Performance Measuring
Crabby has two types of probes for measuring website performance:
- **`selenium`**, which uses the Selenium API to conduct browser-based performance tests via Chrome/chromedriver.  The `selenium` test is appropriate when performance measurement is the primary concern.  These tests will pull down the page along with all objects included in the page.  Due to limitations of chromedriver, the probe does not support reporting of TLS negotiation time or the HTTP response code 
- **`simple`**, which uses Go's built-in HTTP/2-capable client, `net/http`, to conduct simple HTTP `GET` requests.  These requests measure server performance metrics (including TLS negotiation time for HTTPS) but only pulls down the base URL but *not* objects reference by that page.  Being headless, it cannot measure DOM rendering time.  The `simple` probe is appropriate for measuring app/API availabililty and HTTP connection metrics. 

# Using Crabby

To use it, you edit ```config.yaml``` and specify some URLs that you want it to check, optionally providing some pre-provisioned cookies if the site requires them.  Crabby goes and checks the pages on the specified time interval and collects some basic metrics, which it then sends to all of the configured back-end time series databases, storage services, etc., where they can be charted with a tool like [Grafana](http://grafana.org).
