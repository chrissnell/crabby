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

# Two Types of Performance Measuring
Crabby has two types of probes for measuring website performance:
- **`selenium`**, which uses the Selenium API to conduct browser-based performance tests via Chrome/chromedriver.  The `selenium` test is appropriate when performance measurement is the primary concern.  These tests will pull down the page along with all objects included in the page.  Due to limitations of chromedriver, the probe does not support reporting of TLS negotiation time or the HTTP response code 
- **`simple`**, which uses Go's built-in HTTP/2-capable client, `net/http`, to conduct simple HTTP `GET` requests.  These requests measure server performance metrics (including TLS negotiation time for HTTPS) but only pulls down the base URL but *not* objects reference by that page.  Being headless, it cannot measure DOM rendering time.  The `simple` probe is appropriate for measuring app/API availabililty and HTTP connection metrics. 

# Metrics Delivery
Crabby currently supports two protocols for metrics delivery, Graphite and Datadog:

## Graphite
Crabby speaks the Carbon protocol via TCP or UDP for sending performance metrics to remote Graphite servers.  This is a great way to centrally collect metrics from multiple Crabby POPs.  Using a tool like Grafana, you can consolidate those metrics onto a single dashboard and have a very powerful way of looking at your website performance:

![Multi-POP Performance Graph in Grafana](https://chrissnell.github.io/crabby/images/crabby-multi-site-grafana.png "Four crabby nodes sending metrics to Graphite for display in Grafana")

## Datadog
Crabby supports the sending of performance metrics to Datadog for use in graphical dashboards and alerting.  Using Datadog's anomaly detection capability, you can even configure alerts to trigger when site performance suddenly degrades.  When using the `simple` collector, Crabby can also collect HTTP response codes and send a failed service check to Datadog to trigger an alert if a 400- or 500-series error is detected. 

![Multi-POP Performance Graph in Grafana](https://chrissnell.github.io/crabby/images/crabby-datadog.png "Graphing Crabby metrics in a Datadog dashboard")


# Using Crabby
Crabby is configured by a YAML file that you pass via the `-config` flag.  If you don't pass the `-config` flag, Crabby looks for a `config.yaml` by default.  This config file defines the sites that you want to test, as well as the destinations for the metrics that are generated (Graphite or Datadog or both).  If you're using the `selenium` probe, you'll also need to specify the endpoint hostname and port for the Selenium API server.  
 
## Docker
No doubt, the easiest way to use Crabby is with Docker.  This approach requires no compiling and you don't even have to go through the hassle of setting up Selenium, Chromium, and chromedriver because these are all available in an easy-to-use, all-in-one container.  I always keep the latest version of Crabby available on Docker Hub but if you prefer to build your own images, I've included a `Dockerfile` in this repo and an `entrypoint.sh` to handle the startup.  

There's also a `docker-compose.yml` in the [example/](https://github.com/chrissnell/crabby/tree/master/example) directory to get you started.  By using Docker Compose, connecting Crabby to the Selenium server is really easy and requires no effort on your part.

To use Crabby in Docker, you'll need to mount your `config.yaml` Crabby configuration file into the container and set the `CRABBY_CONFIG` environment variable to the location where you mounted it.  Again, the Docker Compose examples handle this for you so if you're unfamiliar with Docker volumes, I recommend using Compose.


#Crabby Configuration Reference
Crabby is configured by a YAML file that you pass via the `-config` flag (defaults to `config.yaml`).
