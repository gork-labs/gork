package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultParameterAdapter_Path(t *testing.T) {
	adapter := NewDefaultParameterAdapter()
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	// Path extraction should return empty string and false
	// because DefaultParameterAdapter doesn't support path parameters
	value, exists := adapter.Path(req, "id")

	if exists {
		t.Error("expected Path to return false (not supported)")
	}

	if value != "" {
		t.Errorf("expected empty string, got %q", value)
	}
}

func TestDefaultParameterAdapter_Cookie(t *testing.T) {
	adapter := NewDefaultParameterAdapter()

	t.Run("existing cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})

		value, exists := adapter.Cookie(req, "session_id")

		if !exists {
			t.Error("expected cookie to exist")
		}

		if value != "abc123" {
			t.Errorf("expected cookie value 'abc123', got %q", value)
		}
	})

	t.Run("missing cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		value, exists := adapter.Cookie(req, "nonexistent")

		if exists {
			t.Error("expected cookie to not exist")
		}

		if value != "" {
			t.Errorf("expected empty string, got %q", value)
		}
	})

	t.Run("cookie with empty value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "empty_cookie", Value: ""})

		value, exists := adapter.Cookie(req, "empty_cookie")

		if !exists {
			t.Error("expected cookie to exist even with empty value")
		}

		if value != "" {
			t.Errorf("expected empty string, got %q", value)
		}
	})

	t.Run("multiple cookies with same name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "duplicate", Value: "first"})
		req.AddCookie(&http.Cookie{Name: "duplicate", Value: "second"})

		value, exists := adapter.Cookie(req, "duplicate")

		if !exists {
			t.Error("expected cookie to exist")
		}

		// Should return the first cookie value (Go's default behavior)
		if value != "first" {
			t.Errorf("expected first cookie value 'first', got %q", value)
		}
	})
}
