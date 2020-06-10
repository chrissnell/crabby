package main

/*
 Portions of this file were derived from Dave Cheney's httpstat:
 https://github.com/davecheney/httpstat

 His code is licensed as follows:

     MIT License

    Copyright (c) 2016 Dave Cheney

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.

*/

import (
	"context"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

// RunSimpleTest starts an HTTP/HTTPS API test of a site within crabby.  It uses Go's built-in net/http client.
func RunApiTest(ctx context.Context, j Job, storage *Storage, client *http.Client, responses map[string]*JobResponse) {
	for ; !prerequisitesMet(j, responses); {
		log.Printf("Prerequisites not met for %s, going to sleep.\n", j.Name)
		time.Sleep(1 * time.Second)
	}
	log.Printf("Prerequisites met. Running %s.\n", j.Name)

	var method = strings.ToUpper(j.Method)
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequest(method, j.URL, strings.NewReader(j.Body))
	if err != nil {
		log.Printf("unable to create request: %v", err)
		return
	}

	addHeaders(req, j)

	var t0, t1, t2, t3, t4 time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { t0 = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { t1 = time.Now() },
		ConnectStart: func(_, _ string) {
			if t1.IsZero() {
				// connecting to IP
				t1 = time.Now()
			}
		},
		ConnectDone: func(net, addr string, err error) {
			if err != nil {
				log.Printf("unable to connect to host %v: %v", addr, err)
			}
			t2 = time.Now()
		},
		GotConn:              func(_ httptrace.GotConnInfo) { t3 = time.Now() },
		GotFirstResponseByte: func() { t4 = time.Now() },
	}

	// We'll use our Context in this request in case we have to shut down midstream
	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Failed to read response:", err)
		return
	}

	// Send our server response code as an event
	storage.EventDistributor <- j.makeEvent(resp.StatusCode)

	// Save response for use by other jobs
	if _, ok := responses[j.Name]; ok {
		responses[j.Name].Completed = true
		responses[j.Name].Response = resp
		log.Println("Response saved job:", j.Name)
		go waitAndClearResponse(responses, j.Name)
	}

	// Even though we never read the response body, if we don't close it,
	// the http.Transport goroutines will terminate and the app will eventually
	// crash due to OOM
	resp.Body.Close()

	t5 := time.Now() // after read body
	if t0.IsZero() {
		// we skipped DNS
		t0 = t1
	}

	url, err := url.Parse(j.URL)
	if err != nil {
		log.Println("Failed to parse URL:", err)
		return
	}

	switch url.Scheme {
	case "https":
		storage.MetricDistributor <- j.makeMetric("dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("server_connection_duration_milliseconds", t2.Sub(t1).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("tls_handshake_duration_milliseconds", t3.Sub(t2).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000)

	case "http":
		storage.MetricDistributor <- j.makeMetric("dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("server_connection_duration_milliseconds", t3.Sub(t1).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000)
		storage.MetricDistributor <- j.makeMetric("time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000)
	}
}

func waitAndClearResponse(responses map[string]*JobResponse, name string) {
	time.Sleep(5 * time.Second)
	responses[name].Completed = false
}

func addHeaders(req *http.Request, j Job) {
	req.Header = http.Header{}
	if j.Header != nil {
		req.Header = j.Header
	}
	if j.ContentType != "" {
		req.Header.Add("content-type", j.ContentType)
	}
}

func prerequisitesMet(job Job, responses map[string]*JobResponse) bool {
	if job.Type != "api" {
		// Job dependencies are only supported for API tests.
		log.Printf("Warning! Job %s requires other jobs to be completed," +
			" but job dependencies are only supported for API tests!\n", job.Name)
		return true
	}

	for _, dep := range job.Requires {
		if !responses[dep].Completed {
			return false
		}
	}
	return true
}
