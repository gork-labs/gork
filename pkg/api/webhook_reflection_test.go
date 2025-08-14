package api

import (
	"context"
	"testing"
)

// Test webhook handler with custom response structures
type CustomWebhookHandler struct{}

type CustomSuccessResponse struct {
	Status      string `json:"status"`
	ProcessedAt string `json:"processed_at"`
	WebhookID   string `json:"webhook_id"`
}

type CustomErrorResponse struct {
	Status    string `json:"status"`
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

func (h *CustomWebhookHandler) ParseRequest(req WebhookRequest) (WebhookEvent, error) {
	return WebhookEvent{Type: "custom.event", ProviderObject: map[string]interface{}{"test": "data"}}, nil
}

func (h *CustomWebhookHandler) SuccessResponse() interface{} {
	return CustomSuccessResponse{
		Status:      "success",
		ProcessedAt: "2025-08-07T18:22:10Z",
		WebhookID:   "webhook_123",
	}
}

func (h *CustomWebhookHandler) ErrorResponse(err error) interface{} {
	return CustomErrorResponse{
		Status:    "error",
		ErrorCode: 400,
		Message:   err.Error(),
	}
}

func (h *CustomWebhookHandler) GetValidEventTypes() []string { return []string{"custom.event"} }

func (h *CustomWebhookHandler) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "Custom"}
}

// Custom webhook request for testing
type CustomWebhookRequest struct{}

func (CustomWebhookRequest) WebhookRequest() {}

func (r *CustomWebhookRequest) Validate(ctx context.Context) error {
	return nil
}

func TestWebhookReflectionOpenAPIGeneration(t *testing.T) {
	t.Run("reflection extracts custom response types from webhook handler", func(t *testing.T) {
		// Create custom webhook handler
		customHandler := &CustomWebhookHandler{}
		webhookHttpHandler := WebhookHandlerFunc(customHandler)

		// Create route registry
		registry := NewRouteRegistry()

		// Register webhook handler
		mockAdapter := &mockTypedRouterAdapter{}
		_, info := createHandlerFromAny(mockAdapter, webhookHttpHandler)
		info.Method = "POST"
		info.Path = "/webhooks/custom"
		registry.Register(info)

		// Generate OpenAPI spec
		spec := GenerateOpenAPI(registry, WithTitle("Custom Webhook API"), WithVersion("1.0.0"))

		// Verify the spec was generated
		if spec == nil {
			t.Fatal("OpenAPI spec should not be nil")
		}

		// Verify the webhook operation exists
		if spec.Paths == nil {
			t.Fatal("Paths should not be nil")
		}

		webhookPath, exists := spec.Paths["/webhooks/custom"]
		if !exists {
			t.Fatal("Webhook path should exist in OpenAPI spec")
		}

		if webhookPath.Post == nil {
			t.Fatal("POST operation should exist for webhook")
		}

		operation := webhookPath.Post

		// Verify success response (200) contains custom fields
		successResponse, exists := operation.Responses["200"]
		if !exists {
			t.Fatal("200 response should exist")
		}

		if successResponse.Content == nil {
			t.Fatal("Success response should have content")
		}

		jsonContent, exists := successResponse.Content["application/json"]
		if !exists {
			t.Fatal("JSON content should exist for success response")
		}

		if jsonContent.Schema == nil {
			t.Fatal("Schema should exist for success response")
		}

		// The schema might be a reference to components, which is valid OpenAPI
		if jsonContent.Schema.Ref != "" {
			t.Logf("✅ Schema uses component reference: %s", jsonContent.Schema.Ref)

			// Verify the component exists in the spec
			if spec.Components == nil || spec.Components.Schemas == nil {
				t.Fatal("Components schemas should exist when using references")
			}

			// Extract component name from reference (e.g., "#/components/schemas/CustomSuccessResponse" -> "CustomSuccessResponse")
			refName := jsonContent.Schema.Ref
			if len(refName) > len("#/components/schemas/") {
				componentName := refName[len("#/components/schemas/"):]
				componentSchema, exists := spec.Components.Schemas[componentName]
				if !exists {
					t.Fatalf("Component schema %s should exist", componentName)
				}

				// Check the actual component schema properties
				if componentSchema.Properties == nil {
					t.Fatalf("Component schema %s should have properties", componentName)
				}

				// Use the component schema for validation
				jsonContent.Schema = componentSchema
			} else {
				t.Fatalf("Invalid schema reference format: %s", refName)
			}
		}

		// Check if the schema has the custom fields from CustomSuccessResponse
		if jsonContent.Schema.Properties == nil {
			t.Fatalf("Schema properties should exist. Schema: %+v, WebhookHandler: %+v", jsonContent.Schema, info.WebhookHandler)
		}

		// Verify custom success response fields (note: using struct field names, not JSON tags)
		// The schema generator uses struct field names, not JSON tag names
		statusField, exists := jsonContent.Schema.Properties["Status"]
		if !exists {
			t.Error("Expected 'Status' field in success response schema")
		} else if statusField.Type != "string" {
			t.Errorf("Expected Status field to be string, got %s", statusField.Type)
		}

		processedAtField, exists := jsonContent.Schema.Properties["ProcessedAt"]
		if !exists {
			t.Error("Expected 'ProcessedAt' field in success response schema")
		} else if processedAtField.Type != "string" {
			t.Errorf("Expected ProcessedAt field to be string, got %s", processedAtField.Type)
		}

		webhookIdField, exists := jsonContent.Schema.Properties["WebhookID"]
		if !exists {
			t.Error("Expected 'WebhookID' field in success response schema")
		} else if webhookIdField.Type != "string" {
			t.Errorf("Expected WebhookID field to be string, got %s", webhookIdField.Type)
		}

		// Verify error response (400) contains custom fields
		errorResponse, exists := operation.Responses["400"]
		if !exists {
			t.Fatal("400 response should exist")
		}

		if errorResponse.Content == nil {
			t.Fatal("Error response should have content")
		}

		errorJsonContent, exists := errorResponse.Content["application/json"]
		if !exists {
			t.Fatal("JSON content should exist for error response")
		}

		if errorJsonContent.Schema == nil {
			t.Fatal("Schema should exist for error response")
		}

		// Handle component reference for error response too
		if errorJsonContent.Schema.Ref != "" {
			t.Logf("✅ Error schema uses component reference: %s", errorJsonContent.Schema.Ref)

			// Extract component name and get the actual schema
			refName := errorJsonContent.Schema.Ref
			if len(refName) > len("#/components/schemas/") {
				componentName := refName[len("#/components/schemas/"):]
				componentSchema, exists := spec.Components.Schemas[componentName]
				if !exists {
					t.Fatalf("Error component schema %s should exist", componentName)
				}

				// Use the component schema for validation
				errorJsonContent.Schema = componentSchema
			}
		}

		if errorJsonContent.Schema.Properties == nil {
			t.Fatal("Error schema properties should exist")
		}

		// Verify custom error response fields (using struct field names)
		errorStatusField, exists := errorJsonContent.Schema.Properties["Status"]
		if !exists {
			t.Error("Expected 'Status' field in error response schema")
		} else if errorStatusField.Type != "string" {
			t.Errorf("Expected error Status field to be string, got %s", errorStatusField.Type)
		}

		errorCodeField, exists := errorJsonContent.Schema.Properties["ErrorCode"]
		if !exists {
			t.Error("Expected 'ErrorCode' field in error response schema")
		} else if errorCodeField.Type != "integer" {
			t.Errorf("Expected ErrorCode field to be integer, got %s", errorCodeField.Type)
		}

		messageField, exists := errorJsonContent.Schema.Properties["Message"]
		if !exists {
			t.Error("Expected 'Message' field in error response schema")
		} else if messageField.Type != "string" {
			t.Errorf("Expected Message field to be string, got %s", messageField.Type)
		}

		t.Logf("✅ Reflection successfully extracted custom response types from webhook handler")
		t.Logf("   Success response fields: %v", getSchemaFieldNames(jsonContent.Schema))
		t.Logf("   Error response fields: %v", getSchemaFieldNames(errorJsonContent.Schema))
	})
}

// Helper function to extract field names from schema for logging
func getSchemaFieldNames(schema *Schema) []string {
	if schema == nil || schema.Properties == nil {
		return nil
	}

	var fieldNames []string
	for fieldName := range schema.Properties {
		fieldNames = append(fieldNames, fieldName)
	}
	return fieldNames
}
