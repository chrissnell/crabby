package main

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"time"

	"sourcegraph.com/sourcegraph/go-selenium"
)

/*
	Order of occurance for available timing measurements:

	navigationStart -> redirectStart -> redirectEnd -> fetchStart ->
	domainLookupStart -> domainLookupEnd -> connectStart -> connectEnd ->
	requestStart -> responseStart -> responseEnd -> domLoading ->
	domInteractive -> domContentLoaded -> domComplete -> loadEventStart ->
	loadEventEnd
*/

type requestTimings struct {
	navigationStart   float64
	redirectStart     float64
	redirectEnd       float64
	fetchStart        float64
	domainLookupStart float64
	domainLookupEnd   float64
	connectStart      float64
	connectEnd        float64
	requestStart      float64
	responseStart     float64
	responseEnd       float64
	domLoading        float64
	domInteractive    float64
	domContentLoaded  float64
	domComplete       float64
	loadEventStart    float64
	loadEventEnd      float64
}

// requestIntervals holds intervals that we derive from requestTimings
type requestIntervals struct {
	dnsDuration              float64
	serverConnectionDuration float64
	serverProcessingDuration float64
	serverResponseDuration   float64
	domRenderingDuration     float64
}

// webRequest is a single test against a web server
type webRequest struct {
	url string
	rt  *requestTimings
	ri  *requestIntervals
	wd  selenium.WebDriver
}

// RunTest sends a Selenium job to the Selenium service for running and
// calculates timings
func RunTest(j Job, seleniumServer string, storage *Storage) error {
	var err error

	wr := newWebRequest(j.URL)

	err = wr.setRemote(seleniumServer)
	if err != nil {
		return err
	}

	// There is a security feature with the popular webdrivers (Chrome, Firefox/Gecko,
	// and possibly others) that prevents you from setting cookies in Selenium
	// when the browser is not already on the domain for which the cookies are
	// being set.  To work around this, we need to first load a bogus page on
	// the same domain (anything that generates a 404 is fine) before attempting
	// tos et the cookies.

	// We only need to use this work-around if we have cookies to set
	if len(j.Cookies) > 0 {
		var buf bytes.Buffer
		var u *url.URL

		u, err = url.Parse(j.URL)
		if err != nil {
			return err
		}

		buf.WriteString(u.Scheme)
		buf.WriteString("://")
		buf.WriteString(u.Host)
		buf.WriteString("/selenium-testing-404")

		err = wr.wd.Get(buf.String())
		if err != nil {
			return err
		}

		err = wr.AddCookies(j.Cookies)
		if err != nil {
			return err
		}

	}

	defer wr.wd.Quit()

	err = wr.wd.Get(wr.url)
	if err != nil {
		return fmt.Errorf("Failed to load page: %v", err)
	}

	err = wr.getTimings()
	if err != nil {
		return err
	}

	fmt.Println(j.Name, "DNS time:", wr.ri.dnsDuration)
	storage.MetricDistributor <- makeMetric(j.Name, "dnsDuration", wr.ri.dnsDuration)

	fmt.Println(j.Name, "Connection establishment time", wr.ri.serverConnectionDuration)
	storage.MetricDistributor <- makeMetric(j.Name, "serverConnectionDuration", wr.ri.serverConnectionDuration)

	fmt.Println(j.Name, "Response time:", wr.ri.serverResponseDuration)
	storage.MetricDistributor <- makeMetric(j.Name, "serverResponseDuration", wr.ri.serverResponseDuration)

	fmt.Println(j.Name, "Server processing time:", wr.ri.serverProcessingDuration)
	storage.MetricDistributor <- makeMetric(j.Name, "serverProcessingDuration", wr.ri.serverProcessingDuration)

	fmt.Println(j.Name, "DOM rendering time:", wr.ri.domRenderingDuration)
	storage.MetricDistributor <- makeMetric(j.Name, "domRenderingDuration", wr.ri.domRenderingDuration)

	err = wr.wd.Close()
	if err != nil {
		return err
	}

	return nil

}

func makeMetric(name string, timing string, value float64) Metric {

	m := Metric{
		Name:      fmt.Sprintf("%v.%v", name, timing),
		Value:     value,
		Timestamp: time.Now(),
	}

	return m
}

func newWebRequest(url string) webRequest {
	rt := &requestTimings{}
	ri := &requestIntervals{}

	wr := webRequest{
		url: url,
		rt:  rt,
		ri:  ri,
	}

	return wr
}

func (wr *webRequest) setRemote(remote string) error {
	var err error

	caps := selenium.Capabilities(map[string]interface{}{"browserName": "chrome"})
	wr.wd, err = selenium.NewRemote(caps, remote)

	if err != nil {
		return fmt.Errorf("Failed to open session %v\n", err)
	}

	return nil
}

func (wr *webRequest) fetchTiming(obj string) (float64, error) {
	ss := fmt.Sprint("return window.performance.timing.", obj)
	timing, err := wr.wd.ExecuteScript(ss, nil)
	if err != nil {
		return 0, fmt.Errorf("Could not fetch timing for %v: %v", obj, err)
	}

	if timing == nil {
		log.Println("Could not fetch timing for", obj)
		return 0, nil
	}
	return timing.(float64), nil
}

func (wr *webRequest) getTimings() error {
	var err error

	wr.rt.navigationStart, err = wr.fetchTiming("navigationStart")
	if err != nil {
		return err
	}

	wr.rt.redirectStart, err = wr.fetchTiming("redirectStart")
	if err != nil {
		return err
	}

	wr.rt.redirectEnd, err = wr.fetchTiming("redirectEnd")
	if err != nil {
		return err
	}

	wr.rt.fetchStart, err = wr.fetchTiming("fetchStart")
	if err != nil {
		return err
	}

	wr.rt.domainLookupStart, err = wr.fetchTiming("domainLookupStart")
	if err != nil {
		return err
	}

	wr.rt.domainLookupEnd, err = wr.fetchTiming("domainLookupEnd")
	if err != nil {
		return err
	}

	wr.rt.connectStart, err = wr.fetchTiming("connectStart")
	if err != nil {
		return err
	}

	wr.rt.connectEnd, err = wr.fetchTiming("connectEnd")
	if err != nil {
		return err
	}

	wr.rt.requestStart, err = wr.fetchTiming("requestStart")
	if err != nil {
		return err
	}

	wr.rt.responseStart, err = wr.fetchTiming("responseStart")
	if err != nil {
		return err
	}

	wr.rt.responseEnd, err = wr.fetchTiming("responseEnd")
	if err != nil {
		return err
	}

	wr.rt.domLoading, err = wr.fetchTiming("domLoading")
	if err != nil {
		return err
	}

	wr.rt.domInteractive, err = wr.fetchTiming("domInteractive")
	if err != nil {
		return err
	}

	wr.rt.domContentLoaded, err = wr.fetchTiming("domContentLoaded")
	if err != nil {
		return err
	}

	wr.rt.domComplete, err = wr.fetchTiming("domComplete")
	if err != nil {
		return err
	}

	wr.rt.loadEventStart, err = wr.fetchTiming("loadEventStart")
	if err != nil {
		return err
	}

	wr.rt.loadEventEnd, err = wr.fetchTiming("loadEventEnd")
	if err != nil {
		return err
	}

	wr.calcIntervals()

	return nil
}

func (wr *webRequest) calcIntervals() {
	// dnsDuration:  Time to complete DNS lookup
	// domainLookupStart -> domainLookupEnd
	wr.ri.dnsDuration = wr.rt.domainLookupEnd - wr.rt.domainLookupStart

	// serverConnectionDuration: Time to initiate a TCP connection
	// connectStart -> connectEnd
	wr.ri.serverConnectionDuration = wr.rt.connectEnd - wr.rt.connectStart

	// serverProcessingDuration: Time for the server to process the HTTP request before
	// sending first byte
	// requestStart -> responseStart
	wr.ri.serverProcessingDuration = wr.rt.responseStart - wr.rt.requestStart

	// serverResponseDuration: Time for the server to send the entire response
	// responseStart -> responseEnd
	wr.ri.serverResponseDuration = wr.rt.responseEnd - wr.rt.responseStart

	// domRenderingDuration: Time to rendor the complete DOM
	// domLoading -> domComplete
	wr.ri.domRenderingDuration = wr.rt.domComplete - wr.rt.domLoading
}
