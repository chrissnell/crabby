#Crabby Configuration Reference
Crabby is configured by means of a YAML file that's passed via the `-config` command line parameter when you start Crabby.
##`jobs` - Configuring pages and URLs to test
The top-level `jobs` array holds a list of all of the sites and URLs that Crabby will test.  There are two types of tests that Crabby can conduct, `selenium` and `simple`, and these are discussed in [README.md](/README.md).
Fields:

| Field Name | Description |
| ---------- | ----------- |
| `name`     | The name of this test.  This will be used to name your Graphite/Datadog metrics |
| `type`     | Type of test to conduct.  Can be `selenium` or `simple`. |
| `url`      | The URL to probe.  Currently only HTTP GETs are supported. |
| `interval` | How often crabby will initiate tests. (seconds) |
| `cookies`  | Cookies that will be set before loading this page.  **This only works for `selenium` tests.** |

### `cookies`
The optional `cookies` array holds all cookies to be set for a given job.  **These only apply to `selenium` tests.**  These will be set before loading the page.  Please note that these will generate an additional hit to your site (a 404 URL, intentionally) to work around a Selenium security "feature" that doesn't allow you to set cookies for a site until the browser is already on that site.

| Field Name | Description |
| ---------- | ----------- |
| `name`     | The name of the cookie to be set. |
| `domain`   | The domain or subdomain for which the cookie is valid. |
| `path`     | The path for which the cookie is valid.  Typically `/`. |
| `value`    | The value of the cookie to be set. |
| `secure`  | May be `true` or `false`.  If `true`, cookie will only be sent over HTTPS. |

##`selenium` - Selenium server configuration
The `selenium` dictionary holds a few parameters for the Selenium testing service.

| Field Name | Description |
| ---------- | ----------- |
| `url`     | The URL of the Selenium RESTful API.  Typically *someserver*:4444/wd/hub |
| `job-stagger-offset`   | To avoid launching multiple Selenium jobs at the same time and stressing your crabby server with lots of concurrent browser activity, Crabby staggers the start of the jobs.  If your job has, for example, an interval of 30 seconds, it will be executed every 30 seconds...but, this 30 second interval will not commence at t=0.  Rather, Crabby will choose a random offset for each job that is somewhere between zero and `job-stagger-offset` seconds.  So, if you specify a job `interval` of 30 seconds and a `job-stagger-offset` of 15, Crabby will randomly choose an offset between 0 and 15.  It might choose 7 seconds, in which case your job will execute at t=7s, 37s, 1m7s, 1m37s, and so on... It is recommended that you choose a `job-stagger-offset` that's less than the largest `interval` that you've chosen for your jobs. **TO-DO: Build a better job scheduling algorithm**
|

##`storage` - Metrics handling configuration
The `storage` dictionary holds configuration for the various metrics storage and handling backends.  Currently, two storage backends are supported, `graphite` and Datadog's `dogstatsd`.

###`graphite` - Graphite server configuration

| Field Name | Description |
| ---------- | ----------- |
| `host`     | The hostname for your Graphite server. |
| `port`     | The port that your Graphite server listens on for metrics submission.  Typically 2003. |
| `protocol`     | `tcp` or `udp`.  Defaults to `tcp`. |
| `namespace`    | A prefix to prepend to all of your metric names.  Useful for when you have more than one Crabby server or use your Graphite server for other things.  Example:  `crabby.crabby-nyc-01` |

###`dogstatsd` - Datadog dogstatd configuration

| Field Name | Description |
| ---------- | ----------- |
| `host`     | The hostname for your dogstatsd server.  Typically `localhost` if you're running dd-agent locally |
| `port`     | The port that your dogstatsd server listens on for metrics submission.  Typically 8125. |
| `namespace`    | A prefix to prepend to all of your metric names.  Recommened to keep Crabby metrics from getting mixed in with other Datadog metrics.  Example:  `crabby` |
| `tags`    | A YAML list/array of Datadog tags to apply to all submitted metrics.  If you have more than one Crabby node, it's recommended that you set the hostname of the node as a tag.  Example: `crabby-sfo-01` |

###`prometheus` - Prometheus server configuration

| Field Name | Description |
| ---------- | ----------- |
| `host`     | The hostname for your Prometheus pushgateway server. |
| `port`     | The port that your Prometheus pushgateway listens on for metrics submission.  Typically 9091. |
| `namespace`    | A prefix to prepend to all of your metric names.  This gets mapped into a Prometheus grouping.  If you provide a namespace here, a grouping of `crabby => NAMESPACE` is created.  Otherwise, a default grouping of `collector => HOSTNAME` is created.  Useful for when you have more than one Crabby server.    Example:  `crabby.crabby-nyc-01` |

###`riemann` - Riemann server configuration

| Field Name | Description |
| ---------- | ----------- |
| `host`     | The hostname for your Prometheus pushgateway server. |
| `port`     | The port that your Prometheus pushgateway listens on for metrics submission.  Typically 9091. |
| `namespace`    | A prefix to prepend to all of your metric names.  If you omit `namespace`, "crabby" will be automatically prepended.|
| `tags`    | A YAML list/array of strings to apply as tags to all submitted events.  If you have more than one Crabby node, it's recommended that you set the hostname of the node as a tag.  Example: `crabby-sfo-01` |


## Internal Metrics Reporting
Optionally, Crabby can report metrics about itself to your storage backends, including memory (heap) and goroutine usage.  This is especially useful if you are doing development on Crabby and trying to track down runtime problems.

| Field Name | Description |
| ---------- | ----------- |
| `report-internal-metrics`     | Whether or not to report internal runtime metrics.  `true` or `false`.  Defaults to `false`. |
| `internal-metrics-gathering-interval`     | How often to gather and report these metrics (seconds).  Defaults to 15. |


# Complete Configuration File Example
```yaml
jobs:
 - name: my_site_some_page
   type: selenium
   url:  https://mysite.org/some/page/
   interval: 30
   cookies:
    - name: auth
      domain: mysite.org
      path: /
      value: DEADBEEFC0W123456789
      secure: true
    - name: session
      domain: mysite.org
      path: /
      value: abDijfeiF3290FijEIO30NC9jkqQER
      secure: false
 - name: another_page
   type: simple
   url: https://some-other-site.org/some/page
   interval: 10
selenium:
 url: http://localhost:4444/wd/hub
 job-stagger-offset: 30
storage:
    graphite:
        host:  graphite.mysite.org
        port: 2003
        protocol: tcp
        metric-namespace: crabby
    dogstatsd:
        host: localhost
        port: 8125
        metric-namespace: crabby
        tags:
            - crabby-sfo-1
report-internal-metrics: true
internal-metrics-gathering-interval: 15
```
