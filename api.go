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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// RunSimpleTest starts an HTTP/HTTPS API test of a site within crabby.  It uses Go's built-in net/http client.
func RunApiTest(ctx context.Context, j Job, storage *Storage, client *http.Client) {
	responses := map[string]json.RawMessage{}
	for _, s := range j.Steps {
		runApiTestStep(ctx, s, storage, client, responses)
	}
}

func runApiTestStep(ctx context.Context, j JobStep, storage *Storage, client *http.Client, responses map[string]json.RawMessage) {
	var method = strings.ToUpper(j.Method)
	if method == "" {
		method = http.MethodGet
	}

	body, err := replacePlaceholders(j.Body, responses)
	if err != nil {
		log.Printf("unable to substitute body variables in body: %v", err)
		return
	}

	req, err := http.NewRequest(method, j.URL, strings.NewReader(body))
	if err != nil {
		log.Printf("unable to create request: %v", err)
		return
	}

	if err := addHeaders(req, j, responses); err != nil {
		log.Printf("unable to process headers: %v", err)
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
	storage.EventDistributor <- j.makeEvent(resp.StatusCode)

	// unmarshal response and save into map
	if responses[j.Name], err = ioutil.ReadAll(resp.Body); err != nil {
		log.Println("WARNING!", err)
	} else {
		out, _ := replacePlaceholders(fmt.Sprintf("%s response:\n{{ %s }}\n", j.Name, j.Name), responses)
		log.Println(out)
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


func addHeaders(req *http.Request, j JobStep, responses map[string]json.RawMessage) error {
	req.Header = http.Header{}

	for k, value := range j.Header {
		key := textproto.CanonicalMIMEHeaderKey(k)
		if _, ok := req.Header[key]; !ok {
			req.Header[key] = value
		} else {
			req.Header[key] = append(req.Header[key], value...)
		}
	}

	if j.ContentType != "" {
		req.Header["Content-Type"] = []string{j.ContentType}
	}

	// replace placeholders in header values
	for key := range req.Header {
		for i := range req.Header[key] {
			var err error
			if req.Header[key][i], err = replacePlaceholders(req.Header[key][i], responses); err != nil {
				return err
			}
		}
	}
	return nil
}

// getResponseValue looks at the responses of previous steps for a response value.
// s should be <stepName>.objectKey1.objectkey2...
// e.g. step1 returns a json { "key": { "subkey": "value" } }.
// 		step2 can access this by putting {{ step1.key.subkey }} to obtain "value"
// Note: this function will fail if the key contains "." e.g. { "bad.key": value }
func getResponseValue(s string, m map[string]json.RawMessage) (string, error) {
	split := strings.SplitN(s, ".",2)
	value := m[split[0]]
	if len(split) == 1 {
		return string(value), nil
	}
	var submap map[string]json.RawMessage
	if err := json.Unmarshal(value, &submap); err != nil {
		return "", err
	}
	return getResponseValue(split[1], submap)
}

func replacePlaceholders(s string, m map[string]json.RawMessage) (string, error) {
	re := regexp.MustCompile(`{{ *[^}}]* *}}`)
	vars := re.FindAll([]byte(s), -1)
	varvals := make([]interface{}, len(vars))
	for i, v := range vars {
		key := string(v)
		key = strings.TrimPrefix(key, "{{")
		key = strings.TrimSuffix(key, "}}")
		key = strings.TrimSpace(key)
		var err error
		if varvals[i], err = getResponseValue(key, m); err != nil {
			return "", err
		}
	}
	format := re.ReplaceAll([]byte(s), []byte("%s"))
	return fmt.Sprintf(string(format), varvals...), nil
}
