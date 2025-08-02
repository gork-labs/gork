package stdlib

import "testing"

func TestToNativePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic path scenarios
		{
			name:     "simple path without params",
			input:    "/health",
			expected: "/health",
		},
		{
			name:     "path with single parameter",
			input:    "/users/{id}",
			expected: "/users/{id}",
		},
		{
			name:     "path with multiple parameters",
			input:    "/users/{id}/posts/{postId}",
			expected: "/users/{id}/posts/{postId}",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		
		// Wildcard conversion scenarios
		{
			name:     "path with trailing wildcard",
			input:    "/docs/*",
			expected: "/docs/{rest...}",
		},
		{
			name:     "root wildcard",
			input:    "/*",
			expected: "/{rest...}",
		},
		{
			name:     "complex path with params and trailing wildcard",
			input:    "/api/v1/users/{id}/*",
			expected: "/api/v1/users/{id}/{rest...}",
		},
		{
			name:     "path with multiple trailing slashes and wildcard",
			input:    "/files//*",
			expected: "/files//{rest...}",
		},
		
		// Edge cases - only trailing wildcard should be converted
		{
			name:     "path with wildcard in middle (unchanged)",
			input:    "/api/*/docs",
			expected: "/api/*/docs",
		},
		{
			name:     "only trailing wildcard should be converted",
			input:    "/api/*/users/*",
			expected: "/api/*/users/{rest...}",
		},
		{
			name:     "multiple wildcards, only last one converted",
			input:    "/*/middle/*/end/*",
			expected: "/*/middle/*/end/{rest...}",
		},
		{
			name:     "wildcard not at end should not be converted",
			input:    "/files/*something",
			expected: "/files/*something",
		},
		{
			name:     "path with query-like syntax (should not be converted)",
			input:    "/search?query=*",
			expected: "/search?query=*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toNativePath(tt.input)
			if result != tt.expected {
				t.Errorf("toNativePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
