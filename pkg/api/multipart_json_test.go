package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type multipartItem struct {
	Name  string `gork:"name"`
	Count int    `gork:"count"`
}

type multipartItemsRequest struct {
	Body struct {
		Text  string          `gork:"text"`
		Items []multipartItem `gork:"items"`
		Times []time.Time     `gork:"times"`
		File  File            `gork:"file"`
	}
}

func multipartItemsHandler(context.Context, multipartItemsRequest) error { return nil }

func parseMultipartItems(t *testing.T, parts ...multipartPart) (multipartItemsRequest, error) {
	t.Helper()
	body, contentType := multipartBody(t, parts...)
	r := httptest.NewRequest(http.MethodPost, "/items", body)
	r.Header.Set("Content-Type", contentType)
	var req multipartItemsRequest
	return req, ParseRequest(r, &req)
}

func TestParseRequest_MultipartJSONSliceField(t *testing.T) {
	req, err := parseMultipartItems(t,
		multipartPart{name: "items", data: `{"name":"a","count":1}`},
		multipartPart{name: "text", data: "hello"},
		multipartPart{name: "items", data: `{"name":"b","count":2}`},
	)
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	want := []multipartItem{{Name: "a", Count: 1}, {Name: "b", Count: 2}}
	if !reflect.DeepEqual(req.Body.Items, want) {
		t.Errorf("Items = %+v, want %+v", req.Body.Items, want)
	}
	if req.Body.Text != "hello" {
		t.Errorf("Text = %q", req.Body.Text)
	}
}

func TestParseRequest_MultipartJSONSliceFieldContentType(t *testing.T) {
	body := "--b\r\nContent-Disposition: form-data; name=\"items\"\r\nContent-Type: application/json\r\n\r\n" +
		`{"name":"a","count":1}` + "\r\n" +
		"--b\r\nContent-Disposition: form-data; name=\"items\"\r\n\r\n" +
		`{"name":"b","count":2}` + "\r\n--b--\r\n"
	r := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=b")
	var req multipartItemsRequest
	if err := ParseRequest(r, &req); err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	want := []multipartItem{{Name: "a", Count: 1}, {Name: "b", Count: 2}}
	if !reflect.DeepEqual(req.Body.Items, want) {
		t.Errorf("Items = %+v, want %+v", req.Body.Items, want)
	}
}

func TestParseRequest_MultipartJSONSliceFieldInvalidJSON(t *testing.T) {
	_, err := parseMultipartItems(t,
		multipartPart{name: "items", data: `{"name":"a"}`},
		multipartPart{name: "items", data: `{"name":`},
	)
	if err == nil || !strings.Contains(err.Error(), "failed to decode form field items") {
		t.Errorf("error = %v", err)
	}
}

func TestMultipartJSONSliceFieldInvalidJSONResponse(t *testing.T) {
	httpHandler, _ := createHandlerFromAny(multipartTestAdapter{}, multipartItemsHandler)
	body, contentType := multipartBody(t, multipartPart{name: "items", data: "not json"})
	r := httptest.NewRequest(http.MethodPost, "/items", body)
	r.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	httpHandler(rec, r)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "failed to decode form field items") {
		t.Errorf("response = %d %s", rec.Code, rec.Body.String())
	}
}

func TestParseRequest_MultipartCodecSliceFieldIsNotJSON(t *testing.T) {
	_, err := parseMultipartItems(t, multipartPart{name: "times", data: `{"a":1}`})
	if err == nil || !strings.Contains(err.Error(), "failed to set form field times") {
		t.Errorf("error = %v", err)
	}
}

func TestIsJSONSliceField(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{"slice of structs", reflect.TypeOf([]multipartItem{}), true},
		{"slice of files", fileSliceType, false},
		{"slice of structs with a codec", reflect.TypeOf([]time.Time{}), false},
		{"slice of strings", stringSliceType, false},
		{"struct", reflect.TypeOf(multipartItem{}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJSONSliceField(tt.typ); got != tt.want {
				t.Errorf("isJSONSliceField = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultipartJSONSliceFieldSpec(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Post("/items", multipartItemsHandler)

	spec := GenerateOpenAPI(registry)
	media := spec.Paths["/items"].Post.RequestBody.Content["multipart/form-data"]
	if media == nil || media.Schema.Ref == "" {
		t.Fatalf("expected a multipart/form-data schema reference, got %+v", media)
	}
	schema := spec.Components.Schemas[strings.TrimPrefix(media.Schema.Ref, "#/components/schemas/")]

	items := schema.Properties["items"]
	if items.Type != "array" || items.Items == nil || items.Items.Ref != "#/components/schemas/multipartItem" {
		t.Errorf("items = %+v", items)
	}
	if spec.Components.Schemas["multipartItem"] == nil {
		t.Errorf("schemas = %v", spec.Components.Schemas)
	}
	wantEncoding := map[string]*Encoding{"items": {ContentType: "application/json"}}
	if !reflect.DeepEqual(media.Encoding, wantEncoding) {
		t.Errorf("encoding = %+v, want %+v", media.Encoding, wantEncoding)
	}
}
