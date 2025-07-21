package echo

import "testing"

func TestToNativePath(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"/users/{id}", "/users/:id"},
		{"/docs/*", "/docs/*"},
		{"/foo/bar", "/foo/bar"},
	}

	for _, tt := range tests {
		if got := toNativePath(tt.in); got != tt.out {
			t.Fatalf("toNativePath(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
