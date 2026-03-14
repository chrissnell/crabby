package cookie

import "testing"

func TestHeaderString(t *testing.T) {
	tests := []struct {
		name    string
		cookies []Cookie
		want    string
	}{
		{
			name: "single cookie",
			cookies: []Cookie{
				{Name: "session", Value: "abc123"},
			},
			want: "session=abc123; ",
		},
		{
			name: "multiple cookies",
			cookies: []Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "user", Value: "jane"},
			},
			want: "session=abc123; user=jane; ",
		},
		{
			name:    "empty slice",
			cookies: []Cookie{},
			want:    "",
		},
		{
			name:    "nil slice",
			cookies: nil,
			want:    "",
		},
		{
			name: "cookie with special characters in value",
			cookies: []Cookie{
				{Name: "token", Value: "val=ue&foo"},
			},
			want: "token=val=ue&foo; ",
		},
		{
			name: "cookie with spaces in value",
			cookies: []Cookie{
				{Name: "msg", Value: "hello world"},
			},
			want: `msg="hello world"; `,
		},
		{
			name: "cookie with empty value",
			cookies: []Cookie{
				{Name: "empty", Value: ""},
			},
			want: "empty=; ",
		},
		{
			name: "extra fields ignored in header",
			cookies: []Cookie{
				{Name: "sid", Domain: "example.com", Path: "/app", Value: "xyz", Secure: true, Expiry: 3600},
			},
			want: "sid=xyz; ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HeaderString(tt.cookies)
			if got != tt.want {
				t.Errorf("HeaderString() = %q, want %q", got, tt.want)
			}
		})
	}
}
