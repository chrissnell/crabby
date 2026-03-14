package job

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// StepResponses maps step names to their raw JSON response bodies.
type StepResponses = map[string]json.RawMessage

var placeholderRegex = regexp.MustCompile(`\{\{\s*([\w.]+)\s*\}\}`)

const maxJSONDepth = 20

// TemplateEngine resolves {{ step.field }} placeholders against step responses.
type TemplateEngine struct{}

// Resolve replaces all {{ step.field }} placeholders in template with values
// looked up from responses.
func (te *TemplateEngine) Resolve(template string, responses StepResponses) (string, error) {
	matches := placeholderRegex.FindAllStringSubmatchIndex(template, -1)
	if len(matches) == 0 {
		return template, nil
	}

	var b strings.Builder
	b.Grow(len(template))
	prev := 0

	for _, loc := range matches {
		// loc[0]:loc[1] is the full match, loc[2]:loc[3] is the capture group
		b.WriteString(template[prev:loc[0]])

		key := template[loc[2]:loc[3]]
		val, err := getResponseValue(key, responses, 0)
		if err != nil {
			return "", fmt.Errorf("resolving %q: %w", key, err)
		}
		b.WriteString(val)
		prev = loc[1]
	}
	b.WriteString(template[prev:])

	return b.String(), nil
}

// getResponseValue walks into nested JSON using dot-separated keys.
// depth prevents stack overflow on pathological input.
func getResponseValue(key string, m map[string]json.RawMessage, depth int) (string, error) {
	if depth >= maxJSONDepth {
		return "", fmt.Errorf("max JSON nesting depth (%d) exceeded", maxJSONDepth)
	}

	parts := strings.SplitN(key, ".", 2)
	value, ok := m[parts[0]]
	if !ok {
		return "", fmt.Errorf("step %q not found in responses", parts[0])
	}

	if len(parts) == 1 {
		// Trim surrounding quotes from JSON string values
		s := string(value)
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			var unquoted string
			if err := json.Unmarshal(value, &unquoted); err != nil {
				return s, nil
			}
			return unquoted, nil
		}
		return s, nil
	}

	var submap map[string]json.RawMessage
	if err := json.Unmarshal(value, &submap); err != nil {
		return "", fmt.Errorf("field %q is not a JSON object: %w", parts[0], err)
	}
	return getResponseValue(parts[1], submap, depth+1)
}
