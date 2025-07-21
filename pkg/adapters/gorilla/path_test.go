package gorilla

import "testing"

func TestToNativePath(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"/docs/*", "/docs/{rest:.*}"},
		{"/users/{id}", "/users/{id}"}, // should remain unchanged except wildcard case
	}

	for _, tt := range tests {
		if got := toNativePath(tt.in); got != tt.out {
			t.Fatalf("toNativePath(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
