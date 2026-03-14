package storage

import (
	"sort"
	"testing"
)

func TestMakeDogstatsdTags(t *testing.T) {
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
			name: "single tag",
			tags: map[string]string{"env": "prod"},
			want: []string{"env:prod"},
		},
		{
			name: "multiple tags",
			tags: map[string]string{"env": "prod", "region": "us-east", "service": "api"},
			want: []string{"env:prod", "region:us-east", "service:api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeDogstatsdTags(tt.tags)

			if len(got) != len(tt.want) {
				t.Fatalf("MakeDogstatsdTags() returned %d tags, want %d", len(got), len(tt.want))
			}

			sort.Strings(got)
			sort.Strings(tt.want)
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MakeDogstatsdTags()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
