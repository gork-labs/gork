package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// TestWebhookProcessingTableDriven uses table-driven tests for webhook processing scenarios
func TestWebhookProcessingTableDriven(t *testing.T) {
	tests := []struct {
		name             string
		setupHandler     func() http.HandlerFunc
		setupRequest     func() *http.Request
		expectedStatus   int
		validateResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "ParseRequestError_ErrorBodyReader",
			setupHandler: func() http.HandlerFunc {
				handler := NewTestWebhookHandler("test-secret")
				return WebhookHandlerFunc(handler)
			},
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/webhook", &errorBodyReader{})
				req.Header.Set("X-Test-Signature", "test-secret")
				return req
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "HandlerWithoutEventTypeValidator",
			setupHandler: func() http.HandlerFunc {
				handler := NewSimpleWebhookHandler()
				return WebhookHandlerFunc(handler)
			},
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "data"}`))
				req.Header.Set("X-Signature", "test-sig")
				return req
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "MissingSignatureValidation",
			setupHandler: func() http.HandlerFunc {
				handler := NewTestWebhookHandler("test-secret")
				return WebhookHandlerFunc(handler)
			},
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "data"}`))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "SignatureVerificationFailure",
			setupHandler: func() http.HandlerFunc {
				handler := NewTestWebhookHandler("correct-secret")
				return WebhookHandlerFunc(handler)
			},
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "data"}`))
				req.Header.Set("X-Test-Signature", "wrong-secret")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "ValidWebhookRequest",
			setupHandler: func() http.HandlerFunc {
				handler := NewTestWebhookHandler("test-secret")
				return WebhookHandlerFunc(handler)
			},
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"test": "payment"}`))
				req.Header.Set("X-Test-Signature", "test-secret")
				return req
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpHandler := tt.setupHandler()
			req := tt.setupRequest()
			rec := httptest.NewRecorder()

			httpHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, rec)
			}
		})
	}
}

// TestRouteFilteringTableDriven tests OpenAPI route filtering with table-driven approach
func TestRouteFilteringTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupRoute     func() *RouteInfo
		expectFiltered bool // true means should be filtered (return false)
	}{
		{
			name: "PointerToOpenAPISpec_ShouldFilter",
			setupRoute: func() *RouteInfo {
				specType := reflect.TypeOf(OpenAPISpec{})
				pointerToSpec := reflect.PtrTo(specType)
				return &RouteInfo{ResponseType: pointerToSpec}
			},
			expectFiltered: true,
		},
		{
			name: "CustomTypeBasedOnOpenAPISpec_ShouldNotFilter",
			setupRoute: func() *RouteInfo {
				type CustomOpenAPISpec OpenAPISpec
				customType := reflect.TypeOf((*CustomOpenAPISpec)(nil))
				return &RouteInfo{ResponseType: customType}
			},
			expectFiltered: false,
		},
		{
			name: "NilResponseType_ShouldNotFilter",
			setupRoute: func() *RouteInfo {
				return &RouteInfo{ResponseType: nil}
			},
			expectFiltered: false,
		},
		{
			name: "NonPointerOpenAPISpec_ShouldNotFilter",
			setupRoute: func() *RouteInfo {
				return &RouteInfo{ResponseType: reflect.TypeOf(OpenAPISpec{})}
			},
			expectFiltered: false,
		},
		{
			name: "InterfaceType_ShouldNotFilter",
			setupRoute: func() *RouteInfo {
				var i interface{}
				return &RouteInfo{ResponseType: reflect.TypeOf(&i).Elem()}
			},
			expectFiltered: false,
		},
		{
			name: "RegularStructType_ShouldNotFilter",
			setupRoute: func() *RouteInfo {
				type RegularStruct struct {
					Name string
				}
				return &RouteInfo{ResponseType: reflect.TypeOf(RegularStruct{})}
			},
			expectFiltered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.setupRoute()
			result := defaultRouteFilter(route)

			if tt.expectFiltered && result {
				t.Error("Expected route to be filtered (return false) but it wasn't")
			}
			if !tt.expectFiltered && !result {
				t.Error("Expected route to NOT be filtered (return true) but it was")
			}
		})
	}
}

// errorBodyReader that causes parsing errors
type errorBodyReader struct{}

func (e *errorBodyReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// Test webhook request without EventTypeValidator interface
type SimpleWebhookRequest struct {
	Headers struct {
		Signature string `gork:"X-Signature" validate:"required"`
	}
	Body []byte
}

func (SimpleWebhookRequest) WebhookRequest() {}

// Simple webhook handler without EventTypeValidator interface
type SimpleWebhookHandler struct{}

func NewSimpleWebhookHandler() WebhookHandler[SimpleWebhookRequest] { return &SimpleWebhookHandler{} }

func (h *SimpleWebhookHandler) ParseRequest(req SimpleWebhookRequest) (WebhookEvent, error) {
	if req.Headers.Signature == "" {
		return WebhookEvent{}, fmt.Errorf("missing signature")
	}
	return WebhookEvent{Type: "test.event", ProviderObject: map[string]string{"data": "test"}}, nil
}

func (h *SimpleWebhookHandler) SuccessResponse() interface{} {
	return map[string]bool{"received": true}
}

func (h *SimpleWebhookHandler) ErrorResponse(err error) interface{} {
	return map[string]string{"error": err.Error()}
}

func (h *SimpleWebhookHandler) GetValidEventTypes() []string {
	return []string{"test.event", "payment.succeeded"}
}

func (h *SimpleWebhookHandler) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "Simple"}
}
