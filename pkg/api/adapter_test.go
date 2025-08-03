package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

// TestFunctionNameExtraction consolidates all function name tests
func TestFunctionNameExtraction(t *testing.T) {
	// Test handlers for function name extraction
	handler1 := func(ctx context.Context, req string) (string, error) { return "", nil }
	handler2 := func(ctx context.Context, req int) (int, error) { return 0, nil }

	tests := []struct {
		name     string
		handler  interface{}
		expected string
	}{
		{
			name:     "named function",
			handler:  DummyHandler,
			expected: "DummyHandler",
		},
		{
			name:     "anonymous function 1",
			handler:  handler1,
			expected: "TestFunctionNameExtraction.func1", // Go generates this name
		},
		{
			name:     "anonymous function 2",
			handler:  handler2,
			expected: "TestFunctionNameExtraction.func2",
		},
		{
			name:     "nil handler",
			handler:  nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil handler" {
				// Test that nil handler panics (expected behavior)
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("getFunctionName with nil should panic, but didn't")
					}
				}()
				getFunctionName(tt.handler)
			} else {
				name := getFunctionName(tt.handler)
				if tt.name == "named function" {
					if name != tt.expected {
						t.Errorf("expected %q, got %q", tt.expected, name)
					}
				} else {
					// For anonymous functions, just check it's not empty
					if name == "" {
						t.Errorf("expected non-empty name for anonymous function, got empty string")
					}
				}
			}
		})
	}

	t.Run("function name trimming", func(t *testing.T) {
		// Test trimming of package paths and prefixes
		tests := []struct {
			input    string
			expected string
		}{
			{"github.com/gork-labs/gork/pkg/api.Handler", "Handler"},
			{"main.(*Server).HandleRequest-fm", "HandleRequest-fm"},
			{"SimpleHandler", "SimpleHandler"},
			{"", ""},
		}

		for _, tt := range tests {
			result := trimFunctionName(tt.input)
			if result != tt.expected {
				t.Errorf("trimFunctionName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})
}

// DummyHandler is used to test getFunctionName
func DummyHandler(ctx context.Context, req string) (string, error) {
	return "", nil
}

// TestHandlerOptions consolidates all handler option tests
func TestHandlerOptions(t *testing.T) {
	t.Run("WithTags", func(t *testing.T) {
		tests := []struct {
			name string
			tags []string
			want []string
		}{
			{
				name: "single tag",
				tags: []string{"api"},
				want: []string{"api"},
			},
			{
				name: "multiple tags",
				tags: []string{"api", "users", "v1"},
				want: []string{"api", "users", "v1"},
			},
			{
				name: "empty tags",
				tags: []string{},
				want: []string{},
			},
			{
				name: "duplicate tags",
				tags: []string{"api", "api", "users"},
				want: []string{"api", "api", "users"}, // WithTags doesn't deduplicate
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				option := WithTags(tt.tags...)
				handlerOption := &HandlerOption{}

				option(handlerOption)

				if len(handlerOption.Tags) != len(tt.want) {
					t.Errorf("WithTags() tags length = %d, want %d", len(handlerOption.Tags), len(tt.want))
				}

				for i, tag := range tt.want {
					if handlerOption.Tags[i] != tag {
						t.Errorf("WithTags() tags[%d] = %s, want %s", i, handlerOption.Tags[i], tag)
					}
				}
			})
		}
	})

	t.Run("WithTags append", func(t *testing.T) {
		// Test that WithTags appends to existing tags
		handlerOption := &HandlerOption{
			Tags: []string{"existing"},
		}

		option := WithTags("new1", "new2")
		option(handlerOption)

		expected := []string{"existing", "new1", "new2"}
		if len(handlerOption.Tags) != len(expected) {
			t.Errorf("WithTags() append length = %d, want %d", len(handlerOption.Tags), len(expected))
		}

		for i, tag := range expected {
			if handlerOption.Tags[i] != tag {
				t.Errorf("WithTags() append tags[%d] = %s, want %s", i, handlerOption.Tags[i], tag)
			}
		}
	})

	t.Run("WithBasicAuth", func(t *testing.T) {
		option := WithBasicAuth()
		handlerOption := &HandlerOption{}

		option(handlerOption)

		if len(handlerOption.Security) != 1 {
			t.Errorf("WithBasicAuth() security length = %d, want 1", len(handlerOption.Security))
		}

		security := handlerOption.Security[0]
		if security.Type != "basic" {
			t.Errorf("WithBasicAuth() security type = %s, want basic", security.Type)
		}

		if len(security.Scopes) != 0 {
			t.Errorf("WithBasicAuth() security scopes = %v, want empty", security.Scopes)
		}
	})

	t.Run("WithBearerTokenAuth", func(t *testing.T) {
		tests := []struct {
			name   string
			scopes []string
			want   []string
		}{
			{
				name:   "no scopes",
				scopes: []string{},
				want:   []string{},
			},
			{
				name:   "single scope",
				scopes: []string{"read"},
				want:   []string{"read"},
			},
			{
				name:   "multiple scopes",
				scopes: []string{"read", "write", "admin"},
				want:   []string{"read", "write", "admin"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				option := WithBearerTokenAuth(tt.scopes...)
				handlerOption := &HandlerOption{}

				option(handlerOption)

				if len(handlerOption.Security) != 1 {
					t.Errorf("WithBearerTokenAuth() security length = %d, want 1", len(handlerOption.Security))
				}

				security := handlerOption.Security[0]
				if security.Type != "bearer" {
					t.Errorf("WithBearerTokenAuth() security type = %s, want bearer", security.Type)
				}

				if len(security.Scopes) != len(tt.want) {
					t.Errorf("WithBearerTokenAuth() scopes length = %d, want %d", len(security.Scopes), len(tt.want))
				}

				for i, scope := range tt.want {
					if security.Scopes[i] != scope {
						t.Errorf("WithBearerTokenAuth() scopes[%d] = %s, want %s", i, security.Scopes[i], scope)
					}
				}
			})
		}
	})

	t.Run("WithAPIKeyAuth", func(t *testing.T) {
		option := WithAPIKeyAuth()
		handlerOption := &HandlerOption{}

		option(handlerOption)

		if len(handlerOption.Security) != 1 {
			t.Errorf("WithAPIKeyAuth() security length = %d, want 1", len(handlerOption.Security))
		}

		security := handlerOption.Security[0]
		if security.Type != "apiKey" {
			t.Errorf("WithAPIKeyAuth() security type = %s, want apiKey", security.Type)
		}

		if len(security.Scopes) != 0 {
			t.Errorf("WithAPIKeyAuth() security scopes = %v, want empty", security.Scopes)
		}
	})

	t.Run("multiple security options", func(t *testing.T) {
		// Test combining multiple security options
		handlerOption := &HandlerOption{}

		// Apply multiple security options
		WithBasicAuth()(handlerOption)
		WithBearerTokenAuth("read", "write")(handlerOption)
		WithAPIKeyAuth()(handlerOption)

		if len(handlerOption.Security) != 3 {
			t.Errorf("Multiple security options length = %d, want 3", len(handlerOption.Security))
		}

		// Verify basic auth
		if handlerOption.Security[0].Type != "basic" {
			t.Errorf("Security[0] type = %s, want basic", handlerOption.Security[0].Type)
		}

		// Verify bearer auth
		if handlerOption.Security[1].Type != "bearer" {
			t.Errorf("Security[1] type = %s, want bearer", handlerOption.Security[1].Type)
		}
		if len(handlerOption.Security[1].Scopes) != 2 {
			t.Errorf("Security[1] scopes length = %d, want 2", len(handlerOption.Security[1].Scopes))
		}

		// Verify API key auth
		if handlerOption.Security[2].Type != "apiKey" {
			t.Errorf("Security[2] type = %s, want apiKey", handlerOption.Security[2].Type)
		}
	})

	t.Run("combined options", func(t *testing.T) {
		// Test combining tags and security options
		handlerOption := &HandlerOption{}

		WithTags("api", "v1")(handlerOption)
		WithBasicAuth()(handlerOption)
		WithTags("auth")(handlerOption) // Should append to existing tags

		expectedTags := []string{"api", "v1", "auth"}
		if len(handlerOption.Tags) != len(expectedTags) {
			t.Errorf("Combined options tags length = %d, want %d", len(handlerOption.Tags), len(expectedTags))
		}

		for i, tag := range expectedTags {
			if handlerOption.Tags[i] != tag {
				t.Errorf("Combined options tags[%d] = %s, want %s", i, handlerOption.Tags[i], tag)
			}
		}

		if len(handlerOption.Security) != 1 {
			t.Errorf("Combined options security length = %d, want 1", len(handlerOption.Security))
		}
	})

	t.Run("empty initialization", func(t *testing.T) {
		// Test that HandlerOption initializes with empty slices
		handlerOption := &HandlerOption{}

		if handlerOption.Tags != nil {
			t.Errorf("HandlerOption.Tags should be nil initially, got %v", handlerOption.Tags)
		}

		if handlerOption.Security != nil {
			t.Errorf("HandlerOption.Security should be nil initially, got %v", handlerOption.Security)
		}
	})
}

// TestExtractFunctionNameFromRuntimeWithFunc tests the function name extraction with custom provider
func TestExtractFunctionNameFromRuntimeWithFunc(t *testing.T) {
	t.Run("with nil function provider result", func(t *testing.T) {
		// Mock provider that returns nil (simulates FuncForPC failing)
		mockProvider := func(uintptr) *runtime.Func {
			return nil
		}

		result := extractFunctionNameFromRuntimeWithFunc(DummyHandler, mockProvider)
		if result != "" {
			t.Errorf("Expected empty string for nil function, got %q", result)
		}
	})

	t.Run("with custom function provider", func(t *testing.T) {
		// Test with the real provider to verify the behavior
		result := extractFunctionNameFromRuntimeWithFunc(DummyHandler, runtime.FuncForPC)
		if result == "" {
			t.Error("Expected non-empty result with real function")
		}
		if result != "DummyHandler" {
			t.Errorf("Expected 'DummyHandler', got %q", result)
		}
	})
}

// TestHTTPParameterAdapter consolidates HTTP parameter adapter tests
func TestHTTPParameterAdapter(t *testing.T) {
	adapter := HTTPParameterAdapter{}

	t.Run("Query", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test?name=Alice&age=30&empty=", nil)

		// Test existing parameter
		value, exists := adapter.Query(r, "name")
		if !exists {
			t.Error("Query parameter 'name' should exist")
		}
		if value != "Alice" {
			t.Errorf("Query parameter 'name': got %q, want %q", value, "Alice")
		}

		// Test non-existing parameter
		value, exists = adapter.Query(r, "nonexistent")
		if exists {
			t.Error("Query parameter 'nonexistent' should not exist")
		}
		if value != "" {
			t.Errorf("Non-existing query parameter: got %q, want empty string", value)
		}

		// Test empty parameter
		value, exists = adapter.Query(r, "empty")
		if exists {
			t.Error("Empty query parameter should not exist")
		}
	})

	t.Run("Header", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test", nil)
		r.Header.Set("X-Custom-Header", "custom-value")
		r.Header.Set("Authorization", "Bearer token123")

		// Test existing header
		value, exists := adapter.Header(r, "X-Custom-Header")
		if !exists {
			t.Error("Header 'X-Custom-Header' should exist")
		}
		if value != "custom-value" {
			t.Errorf("Header 'X-Custom-Header': got %q, want %q", value, "custom-value")
		}

		// Test non-existing header
		value, exists = adapter.Header(r, "Non-Existent")
		if exists {
			t.Error("Header 'Non-Existent' should not exist")
		}
		if value != "" {
			t.Errorf("Non-existing header: got %q, want empty string", value)
		}

		// Test case-insensitive header access
		value, exists = adapter.Header(r, "authorization")
		if !exists {
			t.Error("Header 'authorization' should exist (case-insensitive)")
		}
		if value != "Bearer token123" {
			t.Errorf("Header 'authorization': got %q, want %q", value, "Bearer token123")
		}
	})

	t.Run("Cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test", nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
		r.AddCookie(&http.Cookie{Name: "preferences", Value: "dark-mode"})

		// Test existing cookie
		value, exists := adapter.Cookie(r, "session")
		if !exists {
			t.Error("Cookie 'session' should exist")
		}
		if value != "abc123" {
			t.Errorf("Cookie 'session': got %q, want %q", value, "abc123")
		}

		// Test non-existing cookie
		value, exists = adapter.Cookie(r, "nonexistent")
		if exists {
			t.Error("Cookie 'nonexistent' should not exist")
		}
		if value != "" {
			t.Errorf("Non-existing cookie: got %q, want empty string", value)
		}

		// Test another existing cookie
		value, exists = adapter.Cookie(r, "preferences")
		if !exists {
			t.Error("Cookie 'preferences' should exist")
		}
		if value != "dark-mode" {
			t.Errorf("Cookie 'preferences': got %q, want %q", value, "dark-mode")
		}
	})

	t.Run("Path", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test", nil)

		// Test that Path panics as expected
		defer func() {
			if r := recover(); r == nil {
				t.Error("HTTPParameterAdapter.Path should panic")
			}
		}()
		adapter.Path(r, "id")
	})
}

// Benchmark tests for critical functions
func BenchmarkGetFunctionName(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getFunctionName(DummyHandler)
	}
}
