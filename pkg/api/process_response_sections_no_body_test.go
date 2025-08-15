package api

import (
	"reflect"
	"testing"
)

func TestProcessResponseSections_ConventionalResponseWithoutBody(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	t.Run("conventional response with Headers but no Body should return 204 with headers", func(t *testing.T) {
		type ConventionalResponseWithHeaders struct {
			Headers struct {
				Location     string `gork:"Location"`
				CacheControl string `gork:"Cache-Control"`
			}
		}

		respType := reflect.TypeOf(ConventionalResponseWithHeaders{})
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		route := &RouteInfo{
			Method:      "POST",
			Path:        "/test",
			HandlerName: "TestHandler",
		}

		generator.processResponseSections(respType, operation, components, route)

		// Should have 204 response since no Body field
		response, exists := operation.Responses["204"]
		if !exists {
			t.Fatal("Expected 204 response to be created when no Body field")
		}

		// Should not have 200 response
		if _, exists := operation.Responses["200"]; exists {
			t.Error("Should not have 200 response when no Body field")
		}

		// Should have headers processed on the 204 response
		if len(response.Headers) == 0 {
			t.Error("Expected headers to be processed even for 204 response")
		}

		// Should not have content since there's no Body
		if response.Content != nil {
			t.Error("Expected no content when there's no Body field")
		}
	})

	t.Run("conventional response with Cookies but no Body should return 204", func(t *testing.T) {
		type ConventionalResponseWithCookies struct {
			Cookies struct {
				SessionToken string `gork:"session_token"`
			}
		}

		respType := reflect.TypeOf(ConventionalResponseWithCookies{})
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		route := &RouteInfo{
			Method:      "POST",
			Path:        "/test",
			HandlerName: "TestHandler",
		}

		generator.processResponseSections(respType, operation, components, route)

		// Should have 204 response since no Body field
		if _, exists := operation.Responses["204"]; !exists {
			t.Fatal("Expected 204 response to be created when no Body field")
		}

		// Should not have 200 response
		if _, exists := operation.Responses["200"]; exists {
			t.Error("Should not have 200 response when no Body field")
		}
	})

	t.Run("conventional response with Headers and Cookies but no Body should return 204 with headers", func(t *testing.T) {
		type ConventionalResponseWithHeadersAndCookies struct {
			Headers struct {
				Location string `gork:"Location"`
			}
			Cookies struct {
				SessionToken string `gork:"session_token"`
			}
		}

		respType := reflect.TypeOf(ConventionalResponseWithHeadersAndCookies{})
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		route := &RouteInfo{
			Method:      "DELETE",
			Path:        "/test",
			HandlerName: "TestHandler",
		}

		generator.processResponseSections(respType, operation, components, route)

		// Should have 204 response since no Body field
		response, exists := operation.Responses["204"]
		if !exists {
			t.Fatal("Expected 204 response to be created when no Body field")
		}

		// Should not have 200 response
		if _, exists := operation.Responses["200"]; exists {
			t.Error("Should not have 200 response when no Body field")
		}

		// Should have headers processed on the 204 response
		if len(response.Headers) == 0 {
			t.Error("Expected headers to be processed even for 204 response")
		}

		// Should not have content since there's no Body
		if response.Content != nil {
			t.Error("Expected no content when there's no Body field")
		}
	})

	t.Run("conventional response with Body should return 200 with content", func(t *testing.T) {
		type ConventionalResponseWithBody struct {
			Body struct {
				ID   string `gork:"id"`
				Name string `gork:"name"`
			}
			Headers struct {
				Location string `gork:"Location"`
			}
		}

		respType := reflect.TypeOf(ConventionalResponseWithBody{})
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		route := &RouteInfo{
			Method:      "POST",
			Path:        "/test",
			HandlerName: "TestHandler",
		}

		generator.processResponseSections(respType, operation, components, route)

		// Should have 200 response
		response, exists := operation.Responses["200"]
		if !exists {
			t.Fatal("Expected 200 response to be created")
		}

		// Should have content since there's a Body
		if response.Content == nil {
			t.Error("Expected content when Body field is present")
		}

		// Should have headers processed
		if len(response.Headers) == 0 {
			t.Error("Expected headers to be processed")
		}
	})

	t.Run("nil response type (error-only handlers) should return 204", func(t *testing.T) {
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		route := &RouteInfo{
			Method:      "DELETE",
			Path:        "/test",
			HandlerName: "ErrorOnlyHandler",
		}

		// Pass nil as response type
		generator.processResponseSections(nil, operation, components, route)

		// Should have 204 response
		response, exists := operation.Responses["204"]
		if !exists {
			t.Fatal("Expected 204 response for nil response type")
		}

		// Should be a No Content response
		if response.Description != "No Content" {
			t.Errorf("Expected 'No Content' description, got: %q", response.Description)
		}

		// Should not have content
		if response.Content != nil {
			t.Error("Expected no content for nil response type")
		}

		// Should not have headers
		if len(response.Headers) > 0 {
			t.Error("Expected no headers for nil response type")
		}
	})

	t.Run("empty struct (no fields) should return 204", func(t *testing.T) {
		type EmptyResponse struct{}

		respType := reflect.TypeOf(EmptyResponse{})
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		route := &RouteInfo{
			Method:      "POST",
			Path:        "/test",
			HandlerName: "EmptyResponseHandler",
		}

		generator.processResponseSections(respType, operation, components, route)

		// Should have 204 response
		response, exists := operation.Responses["204"]
		if !exists {
			t.Fatal("Expected 204 response for empty struct")
		}

		// Should be a No Content response
		if response.Description != "No Content" {
			t.Errorf("Expected 'No Content' description, got: %q", response.Description)
		}

		// Should not have content
		if response.Content != nil {
			t.Error("Expected no content for empty struct")
		}

		// Should not have headers
		if len(response.Headers) > 0 {
			t.Error("Expected no headers for empty struct")
		}
	})
}
