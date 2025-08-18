package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// failingReader implements io.ReadCloser and always returns an error
type failingReader struct {
	error
}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, f.error
}

func (f *failingReader) Close() error {
	return nil
}

func TestParseRawBodyField_ErrorHandling(t *testing.T) {
	parser := NewConventionParser()

	t.Run("io.ReadAll error should be returned", func(t *testing.T) {
		// Create a request with a body that will error when read
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)

		// Replace the body with a failing reader
		expectedError := errors.New("mock read error")
		req.Body = &failingReader{error: expectedError}

		// Create a []byte field to set
		var bodyBytes []byte
		bodyValue := reflect.ValueOf(&bodyBytes).Elem()

		err := parser.parseRawBodyField(bodyValue, req)

		if err == nil {
			t.Error("expected error when reading body fails")
		}

		if !strings.Contains(err.Error(), "failed to read raw request body") {
			t.Errorf("expected read error message, got %v", err)
		}

		if !strings.Contains(err.Error(), "mock read error") {
			t.Errorf("expected wrapped original error, got %v", err)
		}
	})

	t.Run("nil body should set empty slice", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.Body = nil // Explicitly set to nil

		var bodyBytes []byte
		bodyValue := reflect.ValueOf(&bodyBytes).Elem()

		err := parser.parseRawBodyField(bodyValue, req)
		if err != nil {
			t.Errorf("expected no error for nil body, got %v", err)
		}

		if bodyBytes == nil {
			t.Error("expected empty slice, got nil")
		}

		if len(bodyBytes) != 0 {
			t.Errorf("expected empty slice, got length %d", len(bodyBytes))
		}
	})

	t.Run("successful body read", func(t *testing.T) {
		testContent := `{"event": "test.webhook", "data": {"id": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(testContent))

		var bodyBytes []byte
		bodyValue := reflect.ValueOf(&bodyBytes).Elem()

		err := parser.parseRawBodyField(bodyValue, req)
		if err != nil {
			t.Errorf("expected no error for successful read, got %v", err)
		}

		if string(bodyBytes) != testContent {
			t.Errorf("expected body content %q, got %q", testContent, string(bodyBytes))
		}
	})
}
