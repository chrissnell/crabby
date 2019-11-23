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
	"time"
)

type simpleRequestIntervals struct {
	dnsDuration              float64
	serverConnectionDuration float64
	tlsHandshakeDuration     float64
	serverProcessingDuration float64
	serverResponseDuration   float64
}

// RunSimpleTest starts a simple HTTP/HTTPS test of a site within crabby.  It does
// not use Selenium to perform this test; instead, it uses Go's built-in net/http client.
func RunSimpleTest(ctx context.Context, j Job, storage *Storage, client *http.Client) {

	req, err := http.NewRequest(http.MethodGet, j.URL, nil)
	if err != nil {
		log.Printf("unable to create request: %v", err)
		return
	}

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
	storage.EventDistributor <- makeEvent(j.Name, resp.StatusCode, j.Tags)

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
		storage.MetricDistributor <- makeMetric(j, "dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "server_connection_duration_milliseconds", t2.Sub(t1).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "tls_handshake_duration_milliseconds", t3.Sub(t2).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000)

	case "http":
		storage.MetricDistributor <- makeMetric(j, "dns_duration_milliseconds", t1.Sub(t0).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "server_connection_duration_milliseconds", t3.Sub(t1).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "server_processing_duration_milliseconds", t4.Sub(t3).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "server_response_duration_milliseconds", t5.Sub(t4).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j, "time_to_first_byte_milliseconds", t4.Sub(t0).Seconds()*1000)
	}
}
