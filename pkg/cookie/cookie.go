package cookie

import (
	"net/http"
	"strings"

	"github.com/chromedp/cdproto/network"
)

// Cookie holds cookie data for web requests.
type Cookie struct {
	Name   string `yaml:"name"`
	Domain string `yaml:"domain"`
	Path   string `yaml:"path"`
	Value  string `yaml:"value"`
	Secure bool   `yaml:"secure,omitempty"`
	Expiry uint   `yaml:"expiry,omitempty"`
}

// HeaderString returns the serialization of cookies for use in a Cookie header.
func HeaderString(cookies []Cookie) string {
	var sb strings.Builder
	for _, c := range cookies {
		httpCookie := http.Cookie{
			Name:  c.Name,
			Value: c.Value,
		}
		sb.WriteString(httpCookie.String())
		sb.WriteString("; ")
	}
	return sb.String()
}

// ToCookieParam converts a Cookie to a chromedp network.CookieParam.
func ToCookieParam(c Cookie) network.CookieParam {
	return network.CookieParam{
		Name:   c.Name,
		Value:  c.Value,
		Domain: c.Domain,
		Path:   c.Path,
		Secure: c.Secure,
	}
}
