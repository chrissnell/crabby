package job

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBrowserFactory_Type(t *testing.T) {
	f := &BrowserFactory{}
	if got := f.Type(); got != "browser" {
		t.Errorf("Type() = %q, want %q", got, "browser")
	}
}

func TestBrowserFactory_Create(t *testing.T) {
	input := `
type: browser
name: test-browser
url: https://example.com
interval: 30
tags:
  env: test
`
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(input), &node); err != nil {
		t.Fatal(err)
	}

	f := &BrowserFactory{}
	j, err := f.Create(*node.Content[0], JobOptions{
		GlobalTags: map[string]string{"region": "us-east"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if j.Name() != "test-browser" {
		t.Errorf("Name() = %q, want %q", j.Name(), "test-browser")
	}

	bj := j.(*BrowserJob)
	if bj.config.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", bj.config.URL, "https://example.com")
	}
	if !*bj.config.Headless {
		t.Error("Headless should default to true")
	}
	if bj.tags["env"] != "test" {
		t.Errorf("job tag env = %q, want %q", bj.tags["env"], "test")
	}
	if bj.tags["region"] != "us-east" {
		t.Errorf("global tag region = %q, want %q", bj.tags["region"], "us-east")
	}
}

func TestBrowserFactory_Create_MissingURL(t *testing.T) {
	input := `
type: browser
name: no-url
interval: 30
`
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(input), &node); err != nil {
		t.Fatal(err)
	}

	f := &BrowserFactory{}
	_, err := f.Create(*node.Content[0], JobOptions{})
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestBrowserFactory_Create_HeadlessFalse(t *testing.T) {
	input := `
type: browser
name: visible
url: https://example.com
interval: 10
headless: false
`
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(input), &node); err != nil {
		t.Fatal(err)
	}

	f := &BrowserFactory{}
	j, err := f.Create(*node.Content[0], JobOptions{})
	if err != nil {
		t.Fatal(err)
	}

	bj := j.(*BrowserJob)
	if *bj.config.Headless {
		t.Error("Headless should be false when explicitly set")
	}
}
