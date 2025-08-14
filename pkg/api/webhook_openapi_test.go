package api

import (
	"context"
	"encoding/json"
	"testing"
)

func TestWebhookOpenAPIGeneration(t *testing.T) {
	// Generic webhook generation validation (no provider import)
	t.Run("generic webhook generates correct OpenAPI spec", func(t *testing.T) {
		// Create route registry
		registry := NewRouteRegistry()

		// Create a generic webhook handler (no specific provider)
		genericHandler := &mockWebhookHandler{}
		webhookHttpHandler := WebhookHandlerFunc(genericHandler)

		// Register webhook handler; detection is automatic for WebhookHandlerFunc
		mockAdapter := &mockTypedRouterAdapter{}
		_, info := createHandlerFromAny(mockAdapter, webhookHttpHandler)

		// Manually set route info fields
		info.Method = "POST"
		info.Path = "/webhooks/generic"

		// Register the route
		registry.Register(info)

		// Generate OpenAPI spec
		spec := GenerateOpenAPI(registry)

		// Verify the path exists
		pathItem := spec.Paths["/webhooks/generic"]
		if pathItem == nil {
			t.Fatal("expected /webhooks/generic path to exist")
		}

		// Verify POST operation exists
		operation := pathItem.Post
		if operation == nil {
			t.Fatal("expected POST operation for webhook")
		}

		// Verify extensions show provider info
		if operation.Extensions != nil {
			if prov, ok := operation.Extensions["x-webhook-provider"].(map[string]string); ok {
				if prov["name"] != "Generic" {
					t.Errorf("expected provider name 'Generic', got %v", prov["name"])
				}
			}
		}
	})

	t.Run("non-webhook handlers use regular OpenAPI generation", func(t *testing.T) {
		// Create route registry
		registry := NewRouteRegistry()

		// Create a regular handler (not webhook) with proper Convention response structure
		regularHandler := func(ctx context.Context, req TestRequest) (*TestConventionalResponse, error) {
			return &TestConventionalResponse{
				Body: struct {
					Message string `gork:"message"`
					Success bool   `gork:"success"`
				}{
					Message: "test",
					Success: true,
				},
			}, nil
		}

		// Register regular handler WITHOUT webhook tag
		mockAdapter := &mockTypedRouterAdapter{}
		_, info := createHandlerFromAny(mockAdapter, regularHandler)

		// Manually set route info fields
		info.Method = "POST"
		info.Path = "/api/test"

		// Register the route
		registry.Register(info)

		// Generate OpenAPI spec
		spec := GenerateOpenAPI(registry)

		// Verify the path exists
		pathItem := spec.Paths["/api/test"]
		if pathItem == nil {
			t.Fatal("expected /api/test path to exist")
		}

		// Verify POST operation exists
		operation := pathItem.Post
		if operation == nil {
			t.Fatal("expected POST operation")
		}

		// Verify it's NOT treated as webhook
		hasWebhookTag := false
		for _, tag := range operation.Tags {
			if tag == "webhook" {
				hasWebhookTag = true
				break
			}
		}
		if hasWebhookTag {
			t.Error("regular handler should not have webhook tag")
		}

		// Verify no webhook-specific extensions
		if operation.Extensions != nil {
			if operation.Extensions["x-webhook-provider"] != nil {
				t.Error("regular handler should not have webhook provider extension")
			}
		}
	})
}

// Mock webhook handler for testing
type mockWebhookHandler struct{}

func (h *mockWebhookHandler) ParseRequest(req WebhookRequest) (WebhookEvent, error) {
	return WebhookEvent{Type: "generic_event", ProviderObject: map[string]interface{}{"test": "data"}}, nil
}

func (h *mockWebhookHandler) SuccessResponse() interface{} {
	return map[string]interface{}{"status": "ok"}
}

func (h *mockWebhookHandler) ErrorResponse(err error) interface{} {
	return map[string]interface{}{"error": err.Error()}
}

func (h *mockWebhookHandler) GetValidEventTypes() []string { return []string{"generic_event"} }

func (h *mockWebhookHandler) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "Generic"}
}

func TestWebhookOpenAPIJSONSerialization(t *testing.T) {
	t.Run("webhook OpenAPI spec can be serialized to JSON", func(t *testing.T) {
		// Create route registry with webhook
		registry := NewRouteRegistry()

		// Create webhook handler
		webhookHttpHandler := WebhookHandlerFunc(&mockWebhookHandler{})

		// Register webhook
		mockAdapter := &mockTypedRouterAdapter{}
		_, info := createHandlerFromAny(mockAdapter, webhookHttpHandler)
		info.Method = "POST"
		info.Path = "/webhooks/stripe"
		registry.Register(info)

		// Generate OpenAPI spec
		spec := GenerateOpenAPI(registry, WithTitle("Webhook API"), WithVersion("1.0.0"))

		// Serialize to JSON
		jsonData, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal OpenAPI spec to JSON: %v", err)
		}

		// Verify it contains webhook-specific content
		jsonStr := string(jsonData)

		if !containsSubstring(jsonStr, "/webhooks/stripe") {
			t.Error("expected JSON to contain webhook path")
		}

		if !containsSubstring(jsonStr, "Webhook endpoint") {
			t.Error("expected JSON to contain webhook description")
		}

		if !containsSubstring(jsonStr, "webhook") {
			t.Error("expected JSON to contain webhook tag")
		}

		// Note: Extensions are marked with json:"-" so they won't appear in JSON output
		// This is intentional since OpenAPI extensions need special handling
	})
}

// Use the existing containsSubstring function from convention_openapi_generator_test.go
