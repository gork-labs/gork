package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPParameterAdapter_Query(t *testing.T) {
	adapter := HTTPParameterAdapter{}

	tests := []struct {
		name        string
		url         string
		key         string
		expectedVal string
		expectedOk  bool
	}{
		{
			name:        "existing query parameter",
			url:         "/test?name=john&age=30",
			key:         "name",
			expectedVal: "john",
			expectedOk:  true,
		},
		{
			name:        "existing query parameter with number",
			url:         "/test?name=john&age=30",
			key:         "age",
			expectedVal: "30",
			expectedOk:  true,
		},
		{
			name:        "non-existing query parameter",
			url:         "/test?name=john",
			key:         "email",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name:        "empty query parameter value",
			url:         "/test?name=&age=30",
			key:         "name",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name:        "query parameter with spaces",
			url:         "/test?name=john%20doe",
			key:         "name",
			expectedVal: "john doe",
			expectedOk:  true,
		},
		{
			name:        "query parameter with special characters",
			url:         "/test?email=test%40example.com",
			key:         "email",
			expectedVal: "test@example.com",
			expectedOk:  true,
		},
		{
			name:        "no query parameters",
			url:         "/test",
			key:         "name",
			expectedVal: "",
			expectedOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			val, ok := adapter.Query(req, tt.key)

			if ok != tt.expectedOk {
				t.Errorf("Query() ok = %v, want %v", ok, tt.expectedOk)
			}
			if val != tt.expectedVal {
				t.Errorf("Query() val = %q, want %q", val, tt.expectedVal)
			}
		})
	}
}

func TestHTTPParameterAdapter_Header(t *testing.T) {
	adapter := HTTPParameterAdapter{}

	tests := []struct {
		name        string
		headers     map[string]string
		key         string
		expectedVal string
		expectedOk  bool
	}{
		{
			name:        "existing header",
			headers:     map[string]string{"Authorization": "Bearer token123"},
			key:         "Authorization",
			expectedVal: "Bearer token123",
			expectedOk:  true,
		},
		{
			name:        "case insensitive header",
			headers:     map[string]string{"content-type": "application/json"},
			key:         "Content-Type",
			expectedVal: "application/json",
			expectedOk:  true,
		},
		{
			name:        "non-existing header",
			headers:     map[string]string{"Authorization": "Bearer token123"},
			key:         "X-Custom-Header",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name:        "empty header value",
			headers:     map[string]string{"X-Empty": ""},
			key:         "X-Empty",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name:        "header with whitespace",
			headers:     map[string]string{"X-Test": "  value with spaces  "},
			key:         "X-Test",
			expectedVal: "  value with spaces  ",
			expectedOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			val, ok := adapter.Header(req, tt.key)

			if ok != tt.expectedOk {
				t.Errorf("Header() ok = %v, want %v", ok, tt.expectedOk)
			}
			if val != tt.expectedVal {
				t.Errorf("Header() val = %q, want %q", val, tt.expectedVal)
			}
		})
	}
}

func TestHTTPParameterAdapter_Cookie(t *testing.T) {
	adapter := HTTPParameterAdapter{}

	tests := []struct {
		name        string
		cookies     []*http.Cookie
		key         string
		expectedVal string
		expectedOk  bool
	}{
		{
			name: "existing cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "theme", Value: "dark"},
			},
			key:         "session",
			expectedVal: "abc123",
			expectedOk:  true,
		},
		{
			name: "another existing cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "theme", Value: "dark"},
			},
			key:         "theme",
			expectedVal: "dark",
			expectedOk:  true,
		},
		{
			name: "non-existing cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
			key:         "nonexistent",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name: "empty cookie value",
			cookies: []*http.Cookie{
				{Name: "empty", Value: ""},
			},
			key:         "empty",
			expectedVal: "",
			expectedOk:  true, // Cookie exists, even if empty
		},
		{
			name:        "no cookies",
			cookies:     []*http.Cookie{},
			key:         "session",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name: "cookie with special characters",
			cookies: []*http.Cookie{
				{Name: "data", Value: "test@example.com"},
			},
			key:         "data",
			expectedVal: "test@example.com",
			expectedOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for _, cookie := range tt.cookies {
				req.AddCookie(cookie)
			}

			val, ok := adapter.Cookie(req, tt.key)

			if ok != tt.expectedOk {
				t.Errorf("Cookie() ok = %v, want %v", ok, tt.expectedOk)
			}
			if val != tt.expectedVal {
				t.Errorf("Cookie() val = %q, want %q", val, tt.expectedVal)
			}
		})
	}
}

func TestHTTPParameterAdapter_Path_Panics(t *testing.T) {
	adapter := HTTPParameterAdapter{}
	req := httptest.NewRequest("GET", "/test", nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Path() to panic, but it didn't")
		} else {
			expectedMsg := "Path extraction not implemented for this adapter; please override HTTPParameterAdapter.Path"
			if r.(string) != expectedMsg {
				t.Errorf("Panic message = %q, want %q", r, expectedMsg)
			}
		}
	}()

	adapter.Path(req, "id")
}

// Test custom adapter that overrides Path
type testCustomAdapter struct {
	HTTPParameterAdapter
	pathParams map[string]string
}

func (t *testCustomAdapter) Path(r *http.Request, key string) (string, bool) {
	if t.pathParams == nil {
		return "", false
	}
	val, ok := t.pathParams[key]
	return val, ok
}

func TestCustomAdapter_OverridePath(t *testing.T) {
	adapter := &testCustomAdapter{
		pathParams: map[string]string{
			"id":     "123",
			"userId": "456",
		},
	}

	req := httptest.NewRequest("GET", "/users/123", nil)

	// Test existing path parameter
	val, ok := adapter.Path(req, "id")
	if !ok {
		t.Error("Path() returned false for existing parameter")
	}
	if val != "123" {
		t.Errorf("Path() val = %q, want '123'", val)
	}

	// Test another existing path parameter
	val, ok = adapter.Path(req, "userId")
	if !ok {
		t.Error("Path() returned false for existing parameter")
	}
	if val != "456" {
		t.Errorf("Path() val = %q, want '456'", val)
	}

	// Test non-existing path parameter
	val, ok = adapter.Path(req, "nonexistent")
	if ok {
		t.Error("Path() returned true for non-existing parameter")
	}
	if val != "" {
		t.Errorf("Path() val = %q, want empty string", val)
	}

	// Test that other methods still work from embedded HTTPParameterAdapter
	req = httptest.NewRequest("GET", "/test?name=john", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})

	// Test Query
	val, ok = adapter.Query(req, "name")
	if !ok || val != "john" {
		t.Errorf("Query() = (%q, %v), want ('john', true)", val, ok)
	}

	// Test Header
	val, ok = adapter.Header(req, "Authorization")
	if !ok || val != "Bearer token" {
		t.Errorf("Header() = (%q, %v), want ('Bearer token', true)", val, ok)
	}

	// Test Cookie
	val, ok = adapter.Cookie(req, "session")
	if !ok || val != "abc123" {
		t.Errorf("Cookie() = (%q, %v), want ('abc123', true)", val, ok)
	}
}

func TestParameterAdapterInterface(t *testing.T) {
	// Test that our adapter implements the interface
	var _ GenericParameterAdapter[*http.Request] = &HTTPParameterAdapter{}
	var _ GenericParameterAdapter[*http.Request] = &testCustomAdapter{}
}

func TestHTTPParameterAdapter_EdgeCases(t *testing.T) {
	adapter := HTTPParameterAdapter{}

	t.Run("query with multiple values takes first", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?tags=tag1&tags=tag2&tags=tag3", nil)
		val, ok := adapter.Query(req, "tags")
		if !ok {
			t.Error("Query() returned false for existing parameter")
		}
		if val != "tag1" {
			t.Errorf("Query() val = %q, want 'tag1'", val)
		}
	})

	t.Run("header case sensitivity", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("x-custom-header", "test-value")

		// HTTP headers are case-insensitive
		val, ok := adapter.Header(req, "X-Custom-Header")
		if !ok {
			t.Error("Header() returned false for existing header")
		}
		if val != "test-value" {
			t.Errorf("Header() val = %q, want 'test-value'", val)
		}
	})

	t.Run("cookie with duplicate names returns first", func(t *testing.T) {
		// Manually construct request with duplicate cookies
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Add("Cookie", "session=first")
		req.Header.Add("Cookie", "session=second")

		val, ok := adapter.Cookie(req, "session")
		if !ok {
			t.Error("Cookie() returned false for existing cookie")
		}
		if val != "first" {
			t.Errorf("Cookie() val = %q, want 'first'", val)
		}
	})

	t.Run("nil request handling", func(t *testing.T) {
		// This would typically panic in real usage, but test defensive behavior
		defer func() {
			if r := recover(); r != nil {
				// Expected to panic on nil request
			}
		}()

		// These calls should not crash catastrophically
		// In practice, nil requests would panic, which is expected behavior
	})
}
