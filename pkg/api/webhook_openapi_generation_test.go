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

		t.Run("processWebhookRequestBody handles interface request type", func(t *testing.T) {
			op := &Operation{Responses: map[string]*Response{}}
			comps := &Components{Schemas: map[string]*Schema{}}
			// Use interface type to trigger generic schema branch
			gen.processWebhookRequestBody(reflect.TypeOf((*WebhookRequest)(nil)).Elem(), op, comps)
			if op.RequestBody == nil || op.RequestBody.Content["application/json"].Schema == nil {
				t.Fatal("expected request body schema to be set")
			}
			if op.RequestBody.Content["application/json"].Schema.Type != "object" {
				t.Fatalf("expected generic object schema, got %s", op.RequestBody.Content["application/json"].Schema.Type)
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

// TestWebhookResponseTypeDetection tests webhook response type detection and reflection
func TestWebhookResponseTypeDetection(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	t.Run("getWebhookResponseType with nil handler", func(t *testing.T) {
		result := generator.getWebhookResponseType(nil, "SuccessResponse")
		if result != nil {
			t.Error("Expected nil result for nil handler")
		}
	})

	t.Run("getWebhookResponseType with non-interface return type", func(t *testing.T) {
		handler := &NonInterfaceHandler{}

		result := generator.getWebhookResponseType(handler, "SuccessResponse")

		// Should return the concrete type directly since it's not interface{}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		expectedType := reflect.TypeOf(CustomSuccessResponse{})
		if result != expectedType {
			t.Errorf("Expected %v, got %v", expectedType, result)
		}
	})

	t.Run("getWebhookResponseType with invalid method name", func(t *testing.T) {
		handler := &CustomWebhookHandler{}

		result := generator.getWebhookResponseType(handler, "NonExistentMethod")
		if result != nil {
			t.Error("Expected nil result for non-existent method")
		}
	})

	t.Run("getWebhookResponseType with handler without expected methods", func(t *testing.T) {
		handler := &InvalidHandler{}

		result := generator.getWebhookResponseType(handler, "SuccessResponse")
		if result != nil {
			t.Error("Expected nil result for handler without SuccessResponse method")
		}
	})

	t.Run("getWebhookResponseType ErrorResponse method", func(t *testing.T) {
		handler := &CustomWebhookHandler{}

		result := generator.getWebhookResponseType(handler, "ErrorResponse")

		if result == nil {
			t.Fatal("Expected non-nil result for ErrorResponse")
		}

		expectedType := reflect.TypeOf(CustomErrorResponse{})
		if result != expectedType {
			t.Errorf("Expected %v, got %v", expectedType, result)
		}
	})

	t.Run("getWebhookResponseType with method that returns nil interface", func(t *testing.T) {
		handler := &NilReturnHandler{}

		result := generator.getWebhookResponseType(handler, "SuccessResponse")
		// When method returns nil, we still get interface{} as the static return type
		expectedType := reflect.TypeOf((*interface{})(nil)).Elem()
		if result != expectedType {
			t.Errorf("Expected %v when method returns nil, got %v", expectedType, result)
		}
	})

	t.Run("getWebhookResponseType with method that has wrong signature", func(t *testing.T) {
		handler := &WrongSignatureHandler{}

		result := generator.getWebhookResponseType(handler, "SuccessResponse")
		// Should handle method with wrong parameters
		if result == nil {
			t.Error("Expected some result even with wrong signature")
		}
	})
}

// Handler that returns nil from interface{} methods
type NilReturnHandler struct{}

func (h *NilReturnHandler) SuccessResponse() interface{} {
	return nil
}

func (h *NilReturnHandler) ErrorResponse(err error) interface{} {
	return nil
}

// Handler with wrong method signatures
type WrongSignatureHandler struct{}

func (h *WrongSignatureHandler) SuccessResponse(wrongParam string) interface{} {
	return "wrong signature"
}

func (h *WrongSignatureHandler) ErrorResponse() interface{} {
	return "wrong signature - no error param"
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
