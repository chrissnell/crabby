package job

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPlaceholderRegex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		matches []string
	}{
		{"simple", "{{ login.token }}", []string{"{{ login.token }}"}},
		{"no_spaces", "{{login.token}}", []string{"{{login.token}}"}},
		{"extra_spaces", "{{  login.token  }}", []string{"{{  login.token  }}"}},
		{"multiple", "{{a.b}} and {{c.d}}", []string{"{{a.b}}", "{{c.d}}"}},
		{"no_match_empty_braces", "{{}}", nil},
		{"no_match_single_brace", "{login.token}", nil},
		{"nested_dots", "{{a.b.c.d}}", []string{"{{a.b.c.d}}"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := placeholderRegex.FindAllString(tt.input, -1)
			if len(matches) != len(tt.matches) {
				t.Fatalf("got %d matches %v, want %d %v", len(matches), matches, len(tt.matches), tt.matches)
			}
			for i, m := range matches {
				if m != tt.matches[i] {
					t.Errorf("match[%d] = %q, want %q", i, m, tt.matches[i])
				}
			}
		})
	}
}

func TestTemplateEngine_Resolve(t *testing.T) {
	te := &TemplateEngine{}

	responses := StepResponses{
		"login": json.RawMessage(`{"token":"abc123","user":{"id":42,"name":"test"}}`),
		"fetch": json.RawMessage(`{"items":[1,2,3]}`),
	}

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "no_placeholders",
			template: "plain string",
			want:     "plain string",
		},
		{
			name:     "simple_field",
			template: "Bearer {{ login.token }}",
			want:     "Bearer abc123",
		},
		{
			name:     "nested_field",
			template: "user={{ login.user.id }}",
			want:     "user=42",
		},
		{
			name:     "nested_string_field",
			template: "name={{ login.user.name }}",
			want:     "name=test",
		},
		{
			name:     "multiple_placeholders",
			template: "{{ login.token }}/{{ login.user.id }}",
			want:     "abc123/42",
		},
		{
			name:     "whole_response",
			template: "data={{ fetch.items }}",
			want:     "data=[1,2,3]",
		},
		{
			name:     "missing_step",
			template: "{{ nonexistent.field }}",
			wantErr:  true,
		},
		{
			name:     "missing_nested_field",
			template: "{{ login.missing }}",
			wantErr:  true,
		},
		{
			name:     "field_not_object",
			template: "{{ login.token.sub }}",
			wantErr:  true,
		},
		{
			name:     "percent_in_surrounding_text",
			template: "100% of {{ login.token }}",
			want:     "100% of abc123",
		},
		{
			name:     "percent_s_pattern",
			template: "%s %d {{ login.token }}",
			want:     "%s %d abc123",
		},
		{
			name:     "empty_template",
			template: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := te.Resolve(tt.template, responses)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_Resolve_PercentInResponseData(t *testing.T) {
	te := &TemplateEngine{}
	responses := StepResponses{
		"step1": json.RawMessage(`{"msg":"100% complete","fmt":"%s %d %v"}`),
	}

	got, err := te.Resolve("result: {{ step1.msg }}", responses)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "result: 100% complete" {
		t.Errorf("got %q, want %q", got, "result: 100% complete")
	}

	got, err = te.Resolve("fmt: {{ step1.fmt }}", responses)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fmt: %s %d %v" {
		t.Errorf("got %q, want %q", got, "fmt: %s %d %v")
	}
}

func TestTemplateEngine_Resolve_MalformedJSON(t *testing.T) {
	te := &TemplateEngine{}
	responses := StepResponses{
		"bad": json.RawMessage(`not valid json`),
	}

	// Accessing top-level should return raw value
	got, err := te.Resolve("{{ bad }}", responses)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "not valid json" {
		t.Errorf("got %q, want %q", got, "not valid json")
	}

	// Trying to descend into malformed JSON should error
	_, err = te.Resolve("{{ bad.field }}", responses)
	if err == nil {
		t.Fatal("expected error for descending into malformed JSON")
	}
}

func TestGetResponseValue_MaxDepth(t *testing.T) {
	keys := make([]string, maxJSONDepth+5)
	for i := range keys {
		keys[i] = "a"
	}
	// Build nested JSON: {"a":{"a":{"a":...}}}
	inner := `"leaf"`
	for i := 0; i < maxJSONDepth+5; i++ {
		inner = `{"a":` + inner + `}`
	}

	responses := StepResponses{
		"deep": json.RawMessage(inner),
	}

	key := strings.Join(append([]string{"deep"}, keys...), ".")
	_, err := getResponseValue(key, responses, 0)
	if err == nil {
		t.Fatal("expected max depth error")
	}
	if !strings.Contains(err.Error(), "max JSON nesting depth") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStepNames(t *testing.T) {
	tests := []struct {
		name    string
		steps   []JobStep
		wantErr string
	}{
		{
			name:  "unique_names",
			steps: []JobStep{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		},
		{
			name:    "duplicate_names",
			steps:   []JobStep{{Name: "login"}, {Name: "fetch"}, {Name: "login"}},
			wantErr: "duplicate name",
		},
		{
			name:    "empty_name",
			steps:   []JobStep{{Name: "a"}, {Name: ""}},
			wantErr: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStepNames(tt.steps)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q doesn't contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
