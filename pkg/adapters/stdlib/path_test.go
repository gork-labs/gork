package stdlib

import "testing"

func TestToNativePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
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
			name:     "path with trailing wildcard",
			input:    "/docs/*",
			expected: "/docs/{rest...}",
		},
		{
			name:     "path with wildcard in middle",
			input:    "/api/*/docs",
			expected: "/api/*/docs",
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
			name:     "path with query-like syntax (should not be converted)",
			input:    "/search?query=*",
			expected: "/search?query=*",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "path with multiple trailing slashes and wildcard",
			input:    "/files//*",
			expected: "/files//{rest...}",
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

func TestToNativePathEdgeCases(t *testing.T) {
	// Test that only trailing /* is converted
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "/api/*/users/*",
			expected: "/api/*/users/{rest...}",
			desc:     "only trailing wildcard should be converted",
		},
		{
			input:    "/*/middle/*/end/*",
			expected: "/*/middle/*/end/{rest...}",
			desc:     "multiple wildcards, only last one converted",
		},
		{
			input:    "/files/*something",
			expected: "/files/*something",
			desc:     "wildcard not at end should not be converted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := toNativePath(tc.input)
			if result != tc.expected {
				t.Errorf("toNativePath(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
