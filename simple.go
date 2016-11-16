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

	"golang.org/x/net/http2"
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
func RunSimpleTest(j Job, storage *Storage) error {

	req, err := http.NewRequest("GET", j.URL, nil)
	if err != nil {
		log.Printf("unable to create request: %v", err)
		return err
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
	req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	url, err := url.Parse(j.URL)

	switch url.Scheme {
	case "https":
		// host, _, err := net.SplitHostPort(req.Host)
		// if err != nil {
		// 	host = req.Host
		// }

		// tr.TLSClientConfig = &tls.Config{
		// 	ServerName:         host,
		// 	InsecureSkipVerify: true,
		// }

		// Because we create a custom TLSClientConfig, we have to opt-in to HTTP/2.
		// See https://github.com/golang/go/issues/14275
		err = http2.ConfigureTransport(tr)
		if err != nil {
			log.Println("failed to prepare transport for HTTP/2:", err)
			return err
		}
	}

	client := &http.Client{
		Transport: tr,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Failed to read response:", err)
		return err
	}

	// Send our server response code as an event
	storage.EventDistributor <- makeEvent(j.Name, resp.StatusCode)

	t5 := time.Now() // after read body
	if t0.IsZero() {
		// we skipped DNS
		t0 = t1
	}

	switch url.Scheme {
	case "https":
		storage.MetricDistributor <- makeMetric(j.Name, "dns_duration", t1.Sub(t0).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j.Name, "server_connection_duration", t2.Sub(t0).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j.Name, "tls_handshake_duration", t3.Sub(t2).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j.Name, "server_response_duration", t4.Sub(t3).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j.Name, "server_processing_duration", t5.Sub(t4).Seconds()*1000)

	case "http":
		storage.MetricDistributor <- makeMetric(j.Name, "dns_duration", t1.Sub(t0).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j.Name, "server_connection_duration", t3.Sub(t1).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j.Name, "server_response_duration", t4.Sub(t3).Seconds()*1000)
		storage.MetricDistributor <- makeMetric(j.Name, "server_processing_duration", t5.Sub(t4).Seconds()*1000)
	}

	return nil
}
