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
| `interval` | How often crabby will initiate tests. |
| `cookies`  | Cookies that will be set before loading this page.  **This only works for `selenium` tests.** |

### `cookies`
The `cookies` array holds all cookies to be set for a given job.  **These only apply to `selenium` tests.**  These will be set before loading the page.  Please note that these will generate an additional hit to your site (a 404 URL, intentionally) to work around a Selenium security "feature" that doesn't allow you to set cookies for a site until the browser is already on that site.

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
| `job-stagger-offset`   | To avoid launching multiple Selenium jobs at the same time and stressing your crabby server with lots of concurrent browser activity, Crabby staggers the start of the jobs.  If your job has, for example, an interval of 30 seconds, it will be executed every 30 seconds...but, this 30 second interval will not commence at t=0.  Rather, Crabby will choose a random offset for each job that is somewhere between zero and `job-stagger-offset` seconds.  So, if you specify a job `interval` of 30 seconds and a `job-stagger-offset` of 15, Crabby will randomly choose an offset between 0 and 15.  It might choose 7 seconds, in which case your job will execute at t=7s, 37s, 1m7s, 1m37s, and so on... 

It is recommended that you choose a `job-stagger-offset` that's less than the largest `interval` that you've chosen for your jobs.

**TO-DO: Build a better job scheduling algorithm**
|

selenium:
 url: http://localhost:4444/wd/hub
 job-stagger-offset: 30



```yaml
 - name: my_front_page
   type: selenium
   url:  https://mysite.org/
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
```
##Example Configuration File
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
