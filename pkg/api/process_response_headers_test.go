package api

import (
	"reflect"
	"testing"
)

// TestProcessResponseHeadersComprehensive tests all paths in processResponseHeaders
func TestProcessResponseHeadersComprehensive(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	t.Run("non-struct headersType - early return", func(t *testing.T) {
		response := &Response{
			Headers: make(map[string]*Header),
		}

		// This should trigger lines 247-249 (non-struct early return)
		generator.processResponseHeaders(reflect.TypeOf("string"), response, components)

		// Should not add any headers
		if len(response.Headers) != 0 {
			t.Error("Expected no headers for non-struct type")
		}
	})

	t.Run("struct with no gork tags", func(t *testing.T) {
		type ResponseHeadersWithoutTags struct {
			Field1 string
			Field2 int
		}

		response := &Response{
			Headers: make(map[string]*Header),
		}

		// This should trigger lines 255-257 (continue for empty gork tags)
		generator.processResponseHeaders(reflect.TypeOf(ResponseHeadersWithoutTags{}), response, components)

		// Should not add any headers
		if len(response.Headers) != 0 {
			t.Error("Expected no headers for struct without gork tags")
		}
	})

	t.Run("struct with gork tags", func(t *testing.T) {
		type ResponseHeadersWithTags struct {
			ContentType   string `gork:"content-type"`
			CacheControl  string `gork:"cache-control"`
			XCustomHeader string `gork:"x-custom-header"`
		}

		response := &Response{
			Headers: make(map[string]*Header),
		}

		// This should process all fields with gork tags
		generator.processResponseHeaders(reflect.TypeOf(ResponseHeadersWithTags{}), response, components)

		// Should add headers for all fields
		if len(response.Headers) != 3 {
			t.Errorf("Expected 3 headers, got %d", len(response.Headers))
		}

		// Check specific headers
		if _, exists := response.Headers["content-type"]; !exists {
			t.Error("Expected 'content-type' header")
		}

		if _, exists := response.Headers["cache-control"]; !exists {
			t.Error("Expected 'cache-control' header")
		}

		if _, exists := response.Headers["x-custom-header"]; !exists {
			t.Error("Expected 'x-custom-header' header")
		}

		// Check header properties
		for name, header := range response.Headers {
			if header.Description != "Response header" {
				t.Errorf("Expected header '%s' to have description 'Response header', got '%s'", name, header.Description)
			}

			if header.Schema == nil {
				t.Errorf("Expected header '%s' to have a schema", name)
			}
		}
	})

	t.Run("struct with mixed fields - some with gork tags, some without", func(t *testing.T) {
		type MixedResponseHeadersStruct struct {
			ContentType   string `gork:"content-type"`
			NormalField   string // No gork tag - should be skipped
			Authorization string `gork:"authorization"`
			AnotherField  int    // No gork tag - should be skipped
		}

		response := &Response{
			Headers: make(map[string]*Header),
		}

		generator.processResponseHeaders(reflect.TypeOf(MixedResponseHeadersStruct{}), response, components)

		// Should only process fields with gork tags
		if len(response.Headers) != 2 {
			t.Errorf("Expected 2 headers (only gork-tagged fields), got %d", len(response.Headers))
		}

		// Should have content-type and authorization headers
		if _, exists := response.Headers["content-type"]; !exists {
			t.Error("Expected 'content-type' header")
		}

		if _, exists := response.Headers["authorization"]; !exists {
			t.Error("Expected 'authorization' header")
		}

		// Should not have headers for non-tagged fields
		if _, exists := response.Headers["NormalField"]; exists {
			t.Error("Should not have processed field without gork tag")
		}

		if _, exists := response.Headers["AnotherField"]; exists {
			t.Error("Should not have processed field without gork tag")
		}
	})

	t.Run("struct with various field types", func(t *testing.T) {
		type ResponseHeadersVariousTypes struct {
			StringHeader string  `gork:"string-header"`
			IntHeader    int     `gork:"int-header"`
			BoolHeader   bool    `gork:"bool-header"`
			FloatHeader  float64 `gork:"float-header"`
		}

		response := &Response{
			Headers: make(map[string]*Header),
		}

		generator.processResponseHeaders(reflect.TypeOf(ResponseHeadersVariousTypes{}), response, components)

		// Should process all fields with gork tags
		if len(response.Headers) != 4 {
			t.Errorf("Expected 4 headers, got %d", len(response.Headers))
		}

		// All should have schemas and descriptions
		for name, header := range response.Headers {
			if header.Description != "Response header" {
				t.Errorf("Expected header '%s' to have standard description", name)
			}

			if header.Schema == nil {
				t.Errorf("Expected header '%s' to have a schema", name)
			}
		}
	})

	t.Run("empty struct", func(t *testing.T) {
		type EmptyResponseHeaders struct{}

		response := &Response{
			Headers: make(map[string]*Header),
		}

		generator.processResponseHeaders(reflect.TypeOf(EmptyResponseHeaders{}), response, components)

		// Should not add any headers for empty struct
		if len(response.Headers) != 0 {
			t.Error("Expected no headers for empty struct")
		}
	})
}
