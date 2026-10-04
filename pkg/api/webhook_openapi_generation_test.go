package api

import (
	"reflect"
	"testing"
)

// TestWebhookOpenAPIInternalLogic consolidates tests for internal webhook OpenAPI generation logic,
// including response type detection, fallback handling, and request body processing.
func TestWebhookOpenAPIInternalLogic(t *testing.T) {
	t.Run("GeneratorBranches", func(t *testing.T) {
		gen := &ConventionOpenAPIGenerator{}

		type StripeWebhookRequest struct{}

		t.Run("getWebhookEventTypes falls back by provider when no handler", func(t *testing.T) {
			route := &RouteInfo{RequestType: reflect.TypeOf(StripeWebhookRequest{})}
			events := gen.getWebhookEventTypes(route)
			if len(events) != 0 {
				t.Fatalf("expected no fallback event types, got %v", events)
			}
		})

		t.Run("webhook request processing uses conventional sections", func(t *testing.T) {
			op := &Operation{
				Parameters: []Parameter{},
				Responses:  map[string]*Response{},
			}
			comps := &Components{Schemas: map[string]*Schema{}}

			// Create a mock conventional request structure
			reqType := reflect.TypeOf(struct {
				Body interface{} `json:"body" gork:"body"`
			}{})

			gen.processRequestSections(reqType, op, comps)
			if op.RequestBody == nil || op.RequestBody.Content["application/json"].Schema == nil {
				t.Fatal("expected request body schema to be set")
			}
		})

		t.Run("addWebhookResponses falls back when handler missing", func(t *testing.T) {
			op := &Operation{Responses: map[string]*Response{}}
			comps := &Components{Schemas: map[string]*Schema{}}
			gen.addWebhookResponses(op, comps, &RouteInfo{WebhookHandler: nil})
			if _, ok := op.Responses["200"]; !ok {
				t.Fatal("expected 200 fallback response")
			}
			if _, ok := op.Responses["400"]; !ok {
				t.Fatal("expected 400 fallback response")
			}
		})
	})

	t.Run("ReflectionHandling", func(t *testing.T) {
		generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

		t.Run("addFallbackWebhookResponses covers fallback scenario", func(t *testing.T) {
			operation := &Operation{
				Responses: make(map[string]*Response),
			}
			components := &Components{}

			// Call the fallback method directly
			generator.addFallbackWebhookResponses(operation, components)

			// Verify fallback responses were added
			if _, exists := operation.Responses["200"]; !exists {
				t.Error("Expected fallback success response (200) to be added")
			}

			if _, exists := operation.Responses["400"]; !exists {
				t.Error("Expected fallback error response (400) to be added")
			}
		})

		t.Run("createFallbackSuccessResponse creates proper response", func(t *testing.T) {
			response := generator.createFallbackSuccessResponse()

			if response == nil {
				t.Fatal("Expected non-nil response")
			}

			if response.Description == "" {
				t.Error("Expected non-empty description")
			}

			if response.Content == nil {
				t.Error("Expected content to be set")
			}

			if jsonContent, exists := response.Content["application/json"]; !exists {
				t.Error("Expected JSON content")
			} else if jsonContent.Schema == nil {
				t.Error("Expected schema in JSON content")
			}
		})

		t.Run("createFallbackErrorResponse creates proper response", func(t *testing.T) {
			response := generator.createFallbackErrorResponse()

			if response == nil {
				t.Fatal("Expected non-nil response")
			}

			if response.Description == "" {
				t.Error("Expected non-empty description")
			}

			if response.Content == nil {
				t.Error("Expected content to be set")
			}

			if jsonContent, exists := response.Content["application/json"]; !exists {
				t.Error("Expected JSON content")
			} else if jsonContent.Schema == nil {
				t.Error("Expected schema in JSON content")
			}
		})
	})
}

// Test handler that returns non-interface{} types
type NonInterfaceHandler struct{}

func (h *NonInterfaceHandler) SuccessResponse() CustomSuccessResponse {
	return CustomSuccessResponse{Status: "ok"}
}

func (h *NonInterfaceHandler) ErrorResponse(err error) CustomErrorResponse {
	return CustomErrorResponse{Status: "error", ErrorCode: 400, Message: err.Error()}
}

// Handler that doesn't have the expected methods
type InvalidHandler struct{}

func (h *InvalidHandler) SomeOtherMethod() string {
	return "not a webhook handler"
}

// TestWebhookConventionalSchemas tests that webhooks use conventional request/response schemas
func TestWebhookConventionalSchemas(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())
	components := &Components{Schemas: map[string]*Schema{}}

	t.Run("webhook uses conventional request sections", func(t *testing.T) {
		// Test that webhooks process headers, body, etc. using standard conventional logic
		op := &Operation{
			Parameters: []Parameter{},
			Responses:  map[string]*Response{},
		}

		// Mock conventional request with Headers and Body
		reqType := reflect.TypeOf(struct {
			Body    interface{} `json:"body" gork:"body"`
			Headers struct {
				ContentType string `json:"Content-Type" gork:"content_type" validate:"required"`
			} `json:"headers" gork:"headers"`
		}{})

		generator.processRequestSections(reqType, op, components)

		// Should have processed both body and headers
		if op.RequestBody == nil {
			t.Fatal("Expected request body to be processed")
		}

		// Should have header parameters
		hasHeaderParam := false
		for _, param := range op.Parameters {
			if param.In == "header" {
				hasHeaderParam = true
				break
			}
		}
		if !hasHeaderParam {
			t.Fatal("Expected header parameters to be processed")
		}
	})
}

// TestWebhookResponseGeneration tests webhook response generation scenarios
func TestWebhookResponseGeneration(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	t.Run("addWebhookResponses with nil webhook handler", func(t *testing.T) {
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		components := &Components{}
		route := &RouteInfo{
			WebhookHandler: nil, // No webhook handler
		}

		// Should fall back to basic responses
		generator.addWebhookResponses(operation, components, route)

		// Verify fallback responses were added
		if _, exists := operation.Responses["200"]; !exists {
			t.Error("Expected fallback success response (200) to be added")
		}

		if _, exists := operation.Responses["400"]; !exists {
			t.Error("Expected fallback error response (400) to be added")
		}
	})

	t.Run("addWebhookResponses with handler missing methods", func(t *testing.T) {
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		components := &Components{}
		route := &RouteInfo{
			WebhookHandler: &InvalidHandler{}, // Handler without SuccessResponse/ErrorResponse
		}

		// Should create fallback responses when reflection fails
		generator.addWebhookResponses(operation, components, route)

		// Should still create responses (fallback)
		if _, exists := operation.Responses["200"]; !exists {
			t.Error("Expected fallback success response (200) to be added")
		}

		if _, exists := operation.Responses["400"]; !exists {
			t.Error("Expected fallback error response (400) to be added")
		}
	})

	t.Run("addWebhookResponses with successful reflection", func(t *testing.T) {
		operation := &Operation{
			Responses: make(map[string]*Response),
		}
		components := &Components{
			Schemas: make(map[string]*Schema),
		}
		route := &RouteInfo{
			WebhookHandler: &CustomWebhookHandler{}, // Valid handler
		}

		// Should use reflection to create proper responses
		generator.addWebhookResponses(operation, components, route)

		// Verify responses were added
		if _, exists := operation.Responses["200"]; !exists {
			t.Error("Expected success response (200) to be added")
		}

		if _, exists := operation.Responses["400"]; !exists {
			t.Error("Expected error response (400) to be added")
		}
	})
}
