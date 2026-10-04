package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type timeEchoRequest struct {
	Path struct {
		At time.Time `gork:"at"`
	}
	Query struct {
		Since *time.Time `gork:"since"`
	}
	Headers struct {
		IfModifiedSince time.Time `gork:"If-Modified-Since"`
	}
	Body struct {
		CreatedAt time.Time  `gork:"created_at"`
		DeletedAt *time.Time `gork:"deleted_at"`
	}
}

type timeEchoResponse struct {
	Headers struct {
		LastModified time.Time `gork:"Last-Modified"`
	}
	Body struct {
		At              time.Time  `gork:"at"`
		Since           *time.Time `gork:"since"`
		IfModifiedSince time.Time  `gork:"if_modified_since"`
		CreatedAt       time.Time  `gork:"created_at"`
		DeletedAt       *time.Time `gork:"deleted_at"`
	}
}

func timeEchoHandler(_ context.Context, req timeEchoRequest) (*timeEchoResponse, error) {
	resp := &timeEchoResponse{}
	resp.Headers.LastModified = req.Path.At
	resp.Body.At = req.Path.At
	resp.Body.Since = req.Query.Since
	resp.Body.IfModifiedSince = req.Headers.IfModifiedSince
	resp.Body.CreatedAt = req.Body.CreatedAt
	resp.Body.DeletedAt = req.Body.DeletedAt
	return resp, nil
}

func TestConventionHandler_TimeFieldsUseRFC3339(t *testing.T) {
	adapter := &mockConventionParameterAdapter{
		pathParams:  map[string]string{"at": "2024-01-02T03:04:05+01:00"},
		queryParams: map[string]string{"since": "2024-01-01T00:00:00Z"},
		headers:     map[string]string{"If-Modified-Since": "2023-12-31T23:59:59Z"},
	}
	handler, _ := NewConventionHandlerFactory().CreateHandler(adapter, timeEchoHandler)

	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"created_at":"2024-02-03T04:05:06Z","deleted_at":null}`))
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	want := `{"at":"2024-01-02T03:04:05+01:00","created_at":"2024-02-03T04:05:06Z","deleted_at":null,"if_modified_since":"2023-12-31T23:59:59Z","since":"2024-01-01T00:00:00Z"}`
	if got := rr.Body.String(); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
	if got := rr.Header().Get("Last-Modified"); got != "2024-01-02T03:04:05+01:00" {
		t.Errorf("Last-Modified = %q, want the RFC3339 time", got)
	}
}

func TestConventionHandler_InvalidTimeParameter(t *testing.T) {
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{"at": "yesterday"},
	}
	handler, _ := NewConventionHandlerFactory().CreateHandler(adapter, timeEchoHandler)

	rr := httptest.NewRecorder()
	handler(rr, httptest.NewRequest(http.MethodPost, "/events", nil))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
