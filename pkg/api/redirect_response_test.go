package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type callbackRequest struct {
	Query struct {
		Code  string `gork:"code"`
		State string `gork:"state"`
	}
}

type callbackResponse struct {
	Headers struct {
		Location string `gork:"Location"`
	}
}

type createdResponse struct {
	Body struct {
		ID string `gork:"id"`
	}
}

func callback(_ context.Context, req callbackRequest) (*callbackResponse, error) {
	if req.Query.State != "valid" {
		return nil, NewHTTPError(http.StatusForbidden, "The state is not valid.")
	}
	resp := &callbackResponse{}
	resp.Headers.Location = "/done?code=" + req.Query.Code
	return resp, nil
}

func TestWithStatusResponse(t *testing.T) {
	tests := []struct {
		name         string
		handler      any
		status       int
		target       string
		wantStatus   int
		wantLocation string
		wantBody     string
	}{
		{
			name:         "redirect",
			handler:      callback,
			status:       http.StatusSeeOther,
			target:       "/callback?code=abc&state=valid",
			wantStatus:   http.StatusSeeOther,
			wantLocation: "/done?code=abc",
		},
		{
			name:       "error keeps the error response",
			handler:    callback,
			status:     http.StatusFound,
			target:     "/callback?code=abc&state=old",
			wantStatus: http.StatusForbidden,
			wantBody:   `{"error":"The state is not valid."}`,
		},
		{
			name:       "nil response",
			handler:    func(context.Context, callbackRequest) (*callbackResponse, error) { return nil, nil },
			status:     http.StatusFound,
			target:     "/callback",
			wantStatus: http.StatusFound,
		},
		{
			name:       "error-only handler",
			handler:    func(context.Context, callbackRequest) error { return nil },
			status:     http.StatusAccepted,
			target:     "/callback",
			wantStatus: http.StatusAccepted,
		},
		{
			name: "body",
			handler: func(context.Context, callbackRequest) (*createdResponse, error) {
				resp := &createdResponse{}
				resp.Body.ID = "42"
				return resp, nil
			},
			status:     http.StatusCreated,
			target:     "/callback",
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":"42"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, tt.handler, WithStatus(tt.status))
			rec := httptest.NewRecorder()
			httpHandler(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if got := rec.Header().Get("Content-Type"); tt.wantBody == "" && got != "" {
				t.Errorf("Content-Type = %q, want none for a response without a body", got)
			}
		})
	}
}

func TestWithStatusInSpec(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/callback", callback, WithStatus(http.StatusSeeOther), WithErrorResponses(http.StatusForbidden))
	router.Post("/items", func(context.Context, struct{}) (*createdResponse, error) { return nil, nil }, WithStatus(http.StatusCreated))
	router.Post("/jobs", func(context.Context, struct{}) error { return nil }, WithStatus(http.StatusAccepted))

	spec := GenerateOpenAPI(registry)

	callbackResponses := spec.Paths["/callback"].Get.Responses
	want := &Response{
		Description: "See Other",
		Headers: map[string]*Header{
			"Location": {Description: "Response header", Schema: &Schema{Type: "string"}},
		},
		headersDocType: "github.com/gork-labs/gork/pkg/api.callbackResponse.Headers",
	}
	if got := callbackResponses["303"]; !reflect.DeepEqual(got, want) {
		t.Errorf("response 303 = %+v, want %+v", got, want)
	}
	for _, code := range []string{"200", "204"} {
		if _, ok := callbackResponses[code]; ok {
			t.Errorf("unexpected response %s", code)
		}
	}
	for _, code := range []string{"400", "403", "422", "500"} {
		if _, ok := callbackResponses[code]; !ok {
			t.Errorf("missing error response %s", code)
		}
	}

	created := spec.Paths["/items"].Post.Responses["201"]
	if created == nil || created.Description != "Created" || created.Content["application/json"] == nil {
		t.Errorf("response 201 = %+v, want Created with a JSON body", created)
	}
	if got := spec.Paths["/jobs"].Post.Responses["202"]; got == nil || got.Description != "Accepted" {
		t.Errorf("response 202 = %+v, want Accepted", got)
	}
}
