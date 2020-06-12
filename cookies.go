package main

import (
	"net/http"
	"strings"

	selenium "sourcegraph.com/sourcegraph/go-selenium"
)

// Cookie holds cookie data specified in our job configuration that is
// later translated into a selenium.Cookie for use in web requests
type Cookie struct {
	Name   string `yaml:"name"`
	Domain string `yaml:"domain"`
	Path   string `yaml:"path"`
	Value  string `yaml:"value"`
	Secure bool   `yaml:"secure,omitempty"`
	Expiry uint   `yaml:"expiry,omitempty"`
}

// HeaderString returns the serialization of multiple cookies for use in a Cookie header
func HeaderString(cookies []Cookie) string {
	var sb strings.Builder
	for _, c := range cookies {
		// We're just going to re-use net/http's implementation of a Cookie, since
		// proper validation and serialization of cookie names is very hairy.

		// In a Cookie header, we don't send any cookie metadata.
		httpCookie := http.Cookie{
			Name:  c.Name,
			Value: c.Value,
		}
		sb.WriteString(httpCookie.String())
		sb.WriteString("; ")
	}
	return sb.String()
}

// AddCookies adds cookies to a webRequest in preparation for
// the Selenium run
func (wr *webRequest) AddCookies(cookies []Cookie) error {
	var err error

	// cj.mu.RLock()
	// defer cj.mu.RUnlock()

	if len(cookies) > 0 {
		for _, c := range cookies {
			sc := selenium.Cookie{
				Name:   c.Name,
				Domain: c.Domain,
				Path:   c.Path,
				Value:  c.Value,
				Secure: c.Secure,
				Expiry: c.Expiry,
			}
			err = wr.wd.AddCookie(&sc)
			if err != nil {
				return err
			}

		}
	}

	return nil
}
