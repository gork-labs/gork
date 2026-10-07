package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type imageRequest struct {
	Query struct {
		ID string `gork:"id"`
	}
}

type imageResponse struct {
	Headers struct {
		CacheControl string `gork:"Cache-Control"`
	}
	Cookies struct {
		Seen string `gork:"seen"`
	}
	Body Binary
}

func getImage(_ context.Context, req imageRequest) (*imageResponse, error) {
	if req.Query.ID == "missing" {
		return nil, NewHTTPError(http.StatusNotFound, "The image does not exist.")
	}
	resp := &imageResponse{}
	resp.Headers.CacheControl = "max-age=60"
	resp.Cookies.Seen = "1"
	resp.Body = Binary{ContentType: "image/png", Data: []byte("\x89PNG{\"a\":1}")}
	return resp, nil
}

func TestBinaryResponse(t *testing.T) {
	tests := []struct {
		name       string
		options    []Option
		target     string
		wantStatus int
		wantType   string
		wantBody   string
	}{
		{
			name:       "bytes and content type as they are",
			target:     "/images?id=1",
			wantStatus: http.StatusOK,
			wantType:   "image/png",
			wantBody:   "\x89PNG{\"a\":1}",
		},
		{
			name:       "status",
			options:    []Option{WithStatus(http.StatusCreated)},
			target:     "/images?id=1",
			wantStatus: http.StatusCreated,
			wantType:   "image/png",
			wantBody:   "\x89PNG{\"a\":1}",
		},
		{
			name:       "handler error keeps the JSON error format",
			target:     "/images?id=missing",
			wantStatus: http.StatusNotFound,
			wantType:   "application/json",
			wantBody:   "{\"error\":\"The image does not exist.\"}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, getImage, tt.options...)
			rec := httptest.NewRecorder()
			httpHandler(rec, httptest.NewRequest(http.MethodGet, tt.target, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantType)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestBinaryResponseHeadersAndCookies(t *testing.T) {
	httpHandler, _ := createHandlerFromAny(&HTTPParameterAdapter{}, getImage)
	rec := httptest.NewRecorder()
	httpHandler(rec, httptest.NewRequest(http.MethodGet, "/images?id=1", nil))

	if got := rec.Header().Get("Cache-Control"); got != "max-age=60" {
		t.Errorf("Cache-Control = %q, want max-age=60", got)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "seen" || cookies[0].Value != "1" {
		t.Errorf("cookies = %v, want the cookie seen=1", cookies)
	}
}

func TestBinaryResponseInSpec(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		want    []string
	}{
		{
			name:    "declared media types",
			options: []Option{WithResponseContentTypes("image/png", "image/jpeg")},
			want:    []string{"image/png", "image/jpeg"},
		},
		{
			name: "default media type",
			want: []string{"application/octet-stream"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRouteRegistry()
			router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
			router.Get("/images", getImage, append(tt.options, WithErrorResponses(http.StatusNotFound))...)

			spec := GenerateOpenAPI(registry)

			if spec.OpenAPI != "3.1.0" {
				t.Errorf("openapi = %q, want 3.1.0", spec.OpenAPI)
			}
			response := spec.Paths["/images"].Get.Responses["200"]
			if len(response.Content) != len(tt.want) {
				t.Fatalf("content = %v, want the media types %v", response.Content, tt.want)
			}
			for _, mediaType := range tt.want {
				want := &MediaType{Schema: &Schema{Type: "string", ContentMediaType: mediaType}}
				if got := response.Content[mediaType]; !reflect.DeepEqual(got, want) {
					t.Errorf("content %s = %+v, want %+v", mediaType, got, want)
				}
			}
			if got := spec.Paths["/images"].Get.Responses["404"]; got == nil || got.Content["application/json"] == nil {
				t.Errorf("response 404 = %+v, want a JSON error response", got)
			}
			if got := response.Headers["Cache-Control"]; got == nil {
				t.Errorf("missing the Cache-Control header in the response")
			}
			if got := spec.Paths["/images"].Get.Responses["500"]; got.Ref != "#/components/responses/InternalServerError" {
				t.Errorf("response 500 = %+v, want the standard error response", got)
			}
		})
	}
}
