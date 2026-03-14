package job

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name       string
		jobTags    map[string]string
		globalTags map[string]string
		want       map[string]string
	}{
		{
			name:       "job tags override global tags",
			jobTags:    map[string]string{"env": "staging", "team": "alpha"},
			globalTags: map[string]string{"env": "prod", "dc": "us-east"},
			want:       map[string]string{"env": "staging", "team": "alpha", "dc": "us-east"},
		},
		{
			name:       "both nil returns empty map",
			jobTags:    nil,
			globalTags: nil,
			want:       map[string]string{},
		},
		{
			name:       "global only",
			jobTags:    nil,
			globalTags: map[string]string{"env": "prod", "dc": "us-east"},
			want:       map[string]string{"env": "prod", "dc": "us-east"},
		},
		{
			name:       "job only",
			jobTags:    map[string]string{"env": "staging"},
			globalTags: nil,
			want:       map[string]string{"env": "staging"},
		},
		{
			name:       "both empty maps",
			jobTags:    map[string]string{},
			globalTags: map[string]string{},
			want:       map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeTags(tt.jobTags, tt.globalTags)
			if got == nil {
				t.Fatal("MergeTags returned nil, expected initialized map")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("key %q = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

// mockJob implements Job for testing.
type mockJob struct {
	name     string
	interval time.Duration
}

func (j *mockJob) Name() string            { return j.name }
func (j *mockJob) Interval() time.Duration { return j.interval }
func (j *mockJob) Run(_ context.Context) ([]Metric, []Event, error) {
	return nil, nil, nil
}

// mockFactory implements JobFactory for testing.
type mockFactory struct {
	typeName  string
	createFn  func(cfg yaml.Node, opts JobOptions) (Job, error)
	createErr error
}

func (f *mockFactory) Type() string { return f.typeName }
func (f *mockFactory) Create(cfg yaml.Node, opts JobOptions) (Job, error) {
	if f.createFn != nil {
		return f.createFn(cfg, opts)
	}
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &mockJob{name: f.typeName, interval: 10 * time.Second}, nil
}

func makeYAMLNodes(t *testing.T, docs ...string) []yaml.Node {
	t.Helper()
	nodes := make([]yaml.Node, 0, len(docs))
	for _, doc := range docs {
		var node yaml.Node
		if err := yaml.Unmarshal([]byte(doc), &node); err != nil {
			t.Fatalf("failed to unmarshal yaml %q: %v", doc, err)
		}
		// yaml.Unmarshal wraps in a document node; use the inner mapping node
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			nodes = append(nodes, *node.Content[0])
		} else {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func TestBuildJobs(t *testing.T) {
	tests := []struct {
		name      string
		factories []JobFactory
		yamlDocs  []string
		wantErr   string
		wantJobs  int
	}{
		{
			name: "single registered factory",
			factories: []JobFactory{
				&mockFactory{typeName: "http"},
			},
			yamlDocs: []string{`type: http`},
			wantJobs: 1,
		},
		{
			name: "multiple jobs with same factory",
			factories: []JobFactory{
				&mockFactory{typeName: "http"},
			},
			yamlDocs: []string{`type: http`, `type: http`},
			wantJobs: 2,
		},
		{
			name: "unknown type returns error",
			factories: []JobFactory{
				&mockFactory{typeName: "http"},
			},
			yamlDocs: []string{`type: grpc`},
			wantErr:  `unknown type "grpc"`,
		},
		{
			name:      "missing type returns error",
			factories: []JobFactory{},
			yamlDocs:  []string{`url: https://example.com`},
			wantErr:   "type not specified",
		},
		{
			name: "factory create error propagates",
			factories: []JobFactory{
				&mockFactory{
					typeName:  "http",
					createErr: fmt.Errorf("bad config"),
				},
			},
			yamlDocs: []string{`type: http`},
			wantErr:  "bad config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager(nil)
			for _, f := range tt.factories {
				jm.RegisterFactory(f)
			}

			nodes := makeYAMLNodes(t, tt.yamlDocs...)
			err := jm.BuildJobs(nodes, JobOptions{})

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if got := err.Error(); !containsSubstring(got, tt.wantErr) {
					t.Fatalf("error %q does not contain %q", got, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(jm.jobs) != tt.wantJobs {
				t.Errorf("job count = %d, want %d", len(jm.jobs), tt.wantJobs)
			}
		})
	}
}

func TestRegisterFactory(t *testing.T) {
	jm := NewJobManager(nil)
	f := &mockFactory{typeName: "http"}
	jm.RegisterFactory(f)

	if _, ok := jm.factories["http"]; !ok {
		t.Error("factory not registered under expected key")
	}

	// registering again with same type overwrites
	f2 := &mockFactory{typeName: "http"}
	jm.RegisterFactory(f2)
	if len(jm.factories) != 1 {
		t.Errorf("factory count = %d, want 1 after overwrite", len(jm.factories))
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
