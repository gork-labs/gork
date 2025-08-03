package api

import (
	"reflect"
	"testing"
)

// TestProcessHeadersSectionComprehensive tests all paths in processHeadersSection
func TestProcessHeadersSectionComprehensive(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	t.Run("non-struct sectionType - early return", func(t *testing.T) {
		operation := &Operation{}

		// This should trigger early return for non-struct type
		generator.processHeadersSection(reflect.TypeOf("string"), operation, components)

		// Should not add any parameters
		if len(operation.Parameters) != 0 {
			t.Error("Expected no parameters for non-struct type")
		}
	})

	t.Run("struct with no gork tags", func(t *testing.T) {
		type HeadersWithoutTags struct {
			Field1 string
			Field2 int
		}

		operation := &Operation{}

		// This should skip fields without gork tags
		generator.processHeadersSection(reflect.TypeOf(HeadersWithoutTags{}), operation, components)

		// Should not add any parameters
		if len(operation.Parameters) != 0 {
			t.Error("Expected no parameters for struct without gork tags")
		}
	})

	t.Run("struct with gork tags", func(t *testing.T) {
		type HeadersWithTags struct {
			Authorization string `gork:"authorization" validate:"required"`
			ContentType   string `gork:"content-type"`
		}

		operation := &Operation{}

		// This should process both fields with gork tags
		generator.processHeadersSection(reflect.TypeOf(HeadersWithTags{}), operation, components)

		// Should add parameters for both fields
		if len(operation.Parameters) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(operation.Parameters))
		}

		// Check parameters are header type
		for _, param := range operation.Parameters {
			if param.In != "header" {
				t.Errorf("Expected parameter in 'header', got '%s'", param.In)
			}
		}
	})
}

// TestProcessCookiesSectionComprehensive tests all paths in processCookiesSection
func TestProcessCookiesSectionComprehensive(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	t.Run("non-struct sectionType - early return", func(t *testing.T) {
		operation := &Operation{}

		// This should trigger early return for non-struct type
		generator.processCookiesSection(reflect.TypeOf("string"), operation, components)

		// Should not add any parameters
		if len(operation.Parameters) != 0 {
			t.Error("Expected no parameters for non-struct type")
		}
	})

	t.Run("struct with no gork tags", func(t *testing.T) {
		type CookiesWithoutTags struct {
			Field1 string
			Field2 int
		}

		operation := &Operation{}

		// This should skip fields without gork tags
		generator.processCookiesSection(reflect.TypeOf(CookiesWithoutTags{}), operation, components)

		// Should not add any parameters
		if len(operation.Parameters) != 0 {
			t.Error("Expected no parameters for struct without gork tags")
		}
	})

	t.Run("struct with gork tags", func(t *testing.T) {
		type CookiesWithTags struct {
			SessionID string `gork:"session_id" validate:"required"`
			UserPref  string `gork:"user_pref"`
		}

		operation := &Operation{}

		// This should process both fields with gork tags
		generator.processCookiesSection(reflect.TypeOf(CookiesWithTags{}), operation, components)

		// Should add parameters for both fields
		if len(operation.Parameters) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(operation.Parameters))
		}

		// Check parameters are cookie type
		for _, param := range operation.Parameters {
			if param.In != "cookie" {
				t.Errorf("Expected parameter in 'cookie', got '%s'", param.In)
			}
		}
	})

	t.Run("struct with mixed fields", func(t *testing.T) {
		type MixedCookiesStruct struct {
			SessionID   string `gork:"session_id"`
			NormalField string // No gork tag - should be skipped
			UserToken   string `gork:"user_token"`
		}

		operation := &Operation{}

		generator.processCookiesSection(reflect.TypeOf(MixedCookiesStruct{}), operation, components)

		// Should only process fields with gork tags
		if len(operation.Parameters) != 2 {
			t.Errorf("Expected 2 parameters (only gork-tagged fields), got %d", len(operation.Parameters))
		}

		// Should have session_id and user_token parameters
		paramNames := make(map[string]bool)
		for _, param := range operation.Parameters {
			paramNames[param.Name] = true
		}

		if !paramNames["session_id"] {
			t.Error("Expected 'session_id' parameter")
		}

		if !paramNames["user_token"] {
			t.Error("Expected 'user_token' parameter")
		}

		if paramNames["NormalField"] {
			t.Error("Should not have processed field without gork tag")
		}
	})
}
