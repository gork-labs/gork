package fiber

import (
	"testing"
)

func TestToNativePathConversions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic parameter conversion
		{"/users/{id}", "/users/:id"},
		{"/api/{version}/users/{id}", "/api/:version/users/:id"},

		// Multiple parameters
		{"/users/{userId}/posts/{postId}", "/users/:userId/posts/:postId"},

		// Wildcard handling
		{"/files/*", "/files/*"},
		{"/static/*", "/static/*"},

		// Mixed parameters and wildcards
		{"/api/{version}/files/*", "/api/:version/files/*"},

		// Edge cases
		{"/", "/"},
		{"", ""},
		{"/simple", "/simple"},
		{"/users/{id}/", "/users/:id/"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := toNativePath(test.input)
			if result != test.expected {
				t.Errorf("toNativePath(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestToNativePathCompatibility(t *testing.T) {
	// These should work with Fiber's router
	paths := []string{
		"/users/:id",
		"/api/:version/users/:id",
		"/files/*",
		"/users/:userId/posts/:postId",
	}

	for _, path := range paths {
		// These paths should be valid for Fiber
		// In a real test, we might create a Fiber app and verify the routes can be registered
		result := toNativePath(path)
		if result != path {
			t.Errorf("Path %q should remain unchanged but got %q", path, result)
		}
	}
}
