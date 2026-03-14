package storage

import (
	"sort"
	"testing"

	"github.com/chrissnell/crabby/pkg/config"
)

func TestMakePrometheusLabels(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		want []string // sorted for comparison
	}{
		{
			name: "empty map",
			tags: map[string]string{},
			want: nil,
		},
		{
			name: "nil map",
			tags: nil,
			want: nil,
		},
		{
			name: "single label",
			tags: map[string]string{"env": "prod"},
			want: []string{"env"},
		},
		{
			name: "multiple labels extracts keys only",
			tags: map[string]string{"env": "prod", "region": "us", "job": "web"},
			want: []string{"env", "job", "region"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakePrometheusLabels(tt.tags)

			if len(got) != len(tt.want) {
				t.Fatalf("MakePrometheusLabels() returned %d labels, want %d", len(got), len(tt.want))
			}

			sort.Strings(got)
			sort.Strings(tt.want)
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MakePrometheusLabels()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewPrometheusBackend_namespace_sanitization(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      string
	}{
		{
			name:      "dashes replaced with underscores",
			namespace: "my-app-metrics",
			want:      "my_app_metrics",
		},
		{
			name:      "no dashes unchanged",
			namespace: "myapp",
			want:      "myapp",
		},
		{
			name:      "empty namespace",
			namespace: "",
			want:      "",
		},
		{
			name:      "multiple consecutive dashes",
			namespace: "my--app",
			want:      "my__app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewPrometheusBackend(config.PrometheusConfig{
				ListenAddr: ":9090",
				Namespace:  tt.namespace,
			})
			if b.namespace != tt.want {
				t.Errorf("namespace = %q, want %q", b.namespace, tt.want)
			}
		})
	}
}

func TestNewPrometheusBackend_name(t *testing.T) {
	b := NewPrometheusBackend(config.PrometheusConfig{ListenAddr: ":9090"})
	if b.Name() != "prometheus" {
		t.Errorf("Name() = %q, want %q", b.Name(), "prometheus")
	}
}
