package api

import (
	"net/http"
	"reflect"
	"testing"
)

// Simple http handler used for route-info inference tests
func simpleHTTPHandler(w http.ResponseWriter, r *http.Request) {}

func TestCreateHandlerFromHTTPFunc_Generic(t *testing.T) {
	h, info := createHandlerFromHTTPFunc(simpleHTTPHandler, nil)
	if h == nil || info == nil {
		t.Fatalf("expected non-nil handler and route info")
	}
	if info.HandlerName != "http_handler" {
		t.Fatalf("unexpected HandlerName: %s", info.HandlerName)
	}
	// For generic http handlers, RequestType should be *http.Request
	if info.RequestType != reflect.TypeOf((*http.Request)(nil)) {
		t.Fatalf("unexpected RequestType: %v", info.RequestType)
	}
	if len(info.Options.Tags) == 0 || info.Options.Tags[0] != "http" {
		t.Fatalf("expected default 'http' tag, got %v", info.Options.Tags)
	}
}

// Minimal webhook request type to build a webhook handler
type testWebhookReq struct{ Body []byte }

func (testWebhookReq) WebhookRequest() {}

// Minimal webhook handler implementation for registry-based detection
type testWebhookProvider struct{}

func (testWebhookProvider) ParseRequest(testWebhookReq) (WebhookEvent, error) {
	return WebhookEvent{Type: "x"}, nil
}
func (testWebhookProvider) SuccessResponse() interface{} { return map[string]bool{"ok": true} }
func (testWebhookProvider) ErrorResponse(err error) interface{} {
	return map[string]string{"error": err.Error()}
}
func (testWebhookProvider) GetValidEventTypes() []string { return []string{"x"} }
func (testWebhookProvider) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "Test"}
}

func TestCreateHandlerFromHTTPFunc_Webhook(t *testing.T) {
	// Build a webhook handler and wrap it as http.HandlerFunc
	wh := testWebhookProvider{}
	httpH := WebhookHandlerFunc[testWebhookReq](wh)

	// Now let factory infer route info from the http.HandlerFunc
	_, info := createHandlerFromHTTPFunc(httpH, nil)
	if info == nil {
		t.Fatalf("expected route info for webhook handler")
	}
	// Webhook detection should now set the concrete provider request type
	if info.RequestType != reflect.TypeOf(testWebhookReq{}) {
		t.Fatalf("expected concrete testWebhookReq type, got %v", info.RequestType)
	}
	if info.WebhookHandler == nil {
		t.Fatalf("expected original webhook handler to be recorded in RouteInfo")
	}
	if len(info.Options.Tags) == 0 || info.Options.Tags[0] != "webhooks" {
		t.Fatalf("expected default 'webhooks' tag, got %v", info.Options.Tags)
	}
}

func TestCreateHandlerFromHTTPFunc_WithTagsOverride(t *testing.T) {
	h, info := createHandlerFromHTTPFunc(simpleHTTPHandler, []Option{WithTags("custom")})
	if h == nil || info == nil {
		t.Fatalf("expected non-nil handler and route info")
	}
	if len(info.Options.Tags) != 1 || info.Options.Tags[0] != "custom" {
		t.Fatalf("expected custom tag override, got %v", info.Options.Tags)
	}
}
