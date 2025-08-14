package api

import (
	"context"
	"net/http"
	"testing"
)

func TestWebhookRouteDetection(t *testing.T) {
	t.Run("http handler func built by WebhookHandlerFunc is detected as webhook", func(t *testing.T) {
		// Create a webhook handler using WebhookHandlerFunc (generic mock to avoid provider import)
		webhookHttpHandler := WebhookHandlerFunc(&mockWebhookHandler{})

		// Register it without any special options; detection should be automatic
		mockAdapter := &mockTypedRouterAdapter{}
		httpHandler, info := createHandlerFromAny(mockAdapter, webhookHttpHandler)

		// Verify it was detected as a webhook
		if info == nil {
			t.Fatal("expected RouteInfo to be created")
		}

		hasWebhookTag := false
		for _, tag := range info.Options.Tags {
			if tag == "webhooks" || tag == "webhook" {
				hasWebhookTag = true
				break
			}
		}
		if !hasWebhookTag {
			t.Errorf("expected webhook tag, got tags: %v", info.Options.Tags)
		}

		if httpHandler == nil {
			t.Error("expected http handler to be returned")
		}

		// Verify RequestType is WebhookRequest interface
		if info.RequestType == nil {
			t.Error("expected RequestType to be set")
		}
	})

	t.Run("regular handler is not detected as webhook", func(t *testing.T) {
		regularHandler := func(ctx context.Context, req TestRequest) (*TestResponse, error) {
			return &TestResponse{Message: "test"}, nil
		}

		mockAdapter := &mockTypedRouterAdapter{}
		httpHandler, info := createHandlerFromAny(mockAdapter, regularHandler)

		// Verify it was not detected as a webhook
		if info == nil {
			t.Fatal("expected RouteInfo to be created")
		}

		hasWebhookTag := false
		for _, tag := range info.Options.Tags {
			if tag == "webhook" || tag == "webhooks" {
				hasWebhookTag = true
				break
			}
		}

		if hasWebhookTag {
			t.Error("regular handler should not have webhook tags")
		}

		if httpHandler == nil {
			t.Error("expected http handler to be returned")
		}
	})

	t.Run("http handler func without webhook tag uses generic types", func(t *testing.T) {
		genericHttpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		mockAdapter := &mockTypedRouterAdapter{}
		httpHandler, info := createHandlerFromAny(mockAdapter, genericHttpHandler)

		// Verify it was processed as generic http handler
		if info == nil {
			t.Fatal("expected RouteInfo to be created")
		}

		// Should have http tag by default
		if len(info.Options.Tags) == 0 || info.Options.Tags[0] != "http" {
			t.Errorf("expected http tag, got tags: %v", info.Options.Tags)
		}

		// RequestType should be *http.Request
		if info.RequestType.String() != "*http.Request" {
			t.Errorf("expected *http.Request type, got %v", info.RequestType)
		}

		if httpHandler == nil {
			t.Error("expected http handler to be returned")
		}
	})
}

func TestTypedRouterWebhookRegistration(t *testing.T) {
	t.Run("webhook handler can be registered via normal Register method", func(t *testing.T) {
		registry := NewRouteRegistry()
		mockAdapter := &mockTypedRouterAdapter{}

		var registeredHandler http.HandlerFunc
		var registeredInfo *RouteInfo

		registerFn := func(method, path string, handler http.HandlerFunc, info *RouteInfo) {
			registeredHandler = handler
			registeredInfo = info
		}

		router := NewTypedRouter(
			"test-underlying",
			registry,
			"/api",
			nil,
			mockAdapter,
			registerFn,
		)

		// Create webhook handler
		webhookHttpHandler := WebhookHandlerFunc(&mockWebhookHandler{})

		// Register webhook using regular Post method (no special option needed)
		router.Post("/webhook/generic", webhookHttpHandler)

		// Verify registration
		routes := registry.GetRoutes()
		if len(routes) != 1 {
			t.Fatalf("expected 1 route, got %d", len(routes))
		}

		route := routes[0]
		if route.Method != "POST" {
			t.Errorf("expected POST method, got %s", route.Method)
		}

		if route.Path != "/api/webhook/generic" {
			t.Errorf("expected path '/api/webhook/generic', got %s", route.Path)
		}

		// Check that default webhook tag was assigned
		hasWebhookTag := false
		for _, tag := range route.Options.Tags {
			if tag == "webhooks" || tag == "webhook" {
				hasWebhookTag = true
				break
			}
		}
		if !hasWebhookTag {
			t.Error("expected webhook tag in registered route")
		}

		if registeredHandler == nil {
			t.Error("expected handler to be registered")
		}

		if registeredInfo == nil {
			t.Error("expected route info to be passed to register function")
		}
	})
}
