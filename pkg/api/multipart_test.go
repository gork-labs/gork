package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type multipartRequest struct {
	Path struct {
		ChatID string `gork:"chat_id" validate:"required"`
	}
	Body struct {
		Text   string   `gork:"text" validate:"required,max=10"`
		Count  int      `gork:"count"`
		Pinned bool     `gork:"pinned"`
		Tags   []string `gork:"tags"`
		Avatar File     `gork:"avatar" validate:"required"`
		Images []File   `gork:"images" validate:"min=1,max=2"`
	}
}

type multipartPart struct {
	name, filename, contentType, data string
}

type multipartTestAdapter struct{ HTTPParameterAdapter }

func (multipartTestAdapter) Path(r *http.Request, key string) (string, bool) {
	return r.PathValue(key), true
}

func multipartBody(t *testing.T, parts ...multipartPart) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	for _, p := range parts {
		if p.filename == "" {
			if err := w.WriteField(p.name, p.data); err != nil {
				t.Fatal(err)
			}
			continue
		}
		header := make(map[string][]string)
		header["Content-Disposition"] = []string{`form-data; name="` + p.name + `"; filename="` + p.filename + `"`}
		header["Content-Type"] = []string{p.contentType}
		part, err := w.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(p.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return body, w.FormDataContentType()
}

func parseMultipart(t *testing.T, parts ...multipartPart) (multipartRequest, error) {
	t.Helper()
	body, contentType := multipartBody(t, parts...)
	r := httptest.NewRequest(http.MethodPost, "/chats/1/messages", body)
	r.Header.Set("Content-Type", contentType)
	var req multipartRequest
	return req, ParseRequest(r, &req)
}

func TestParseRequest_MultipartTextFields(t *testing.T) {
	req, err := parseMultipart(t,
		multipartPart{name: "text", data: "hello"},
		multipartPart{name: "count", data: "7"},
		multipartPart{name: "pinned", data: "true"},
		multipartPart{name: "tags", data: "a, b"},
		multipartPart{name: "unknown", data: "ignored"},
	)
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Body.Text != "hello" || req.Body.Count != 7 || !req.Body.Pinned {
		t.Errorf("unexpected text fields: %+v", req.Body)
	}
	if !reflect.DeepEqual(req.Body.Tags, []string{"a", "b"}) {
		t.Errorf("Tags = %v", req.Body.Tags)
	}
}

func TestParseRequest_MultipartOneFile(t *testing.T) {
	req, err := parseMultipart(t, multipartPart{name: "avatar", filename: "me.png", contentType: "image/png", data: "PNG"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	want := File{Name: "me.png", ContentType: "image/png", Data: []byte("PNG")}
	if !reflect.DeepEqual(req.Body.Avatar, want) {
		t.Errorf("Avatar = %+v, want %+v", req.Body.Avatar, want)
	}
	if req.Body.Images != nil {
		t.Errorf("Images = %v, want nil", req.Body.Images)
	}
}

func TestParseRequest_MultipartManyFiles(t *testing.T) {
	req, err := parseMultipart(t,
		multipartPart{name: "images", filename: "a.png", contentType: "image/png", data: "A"},
		multipartPart{name: "images", filename: "b.jpg", contentType: "image/jpeg", data: ""},
	)
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	want := []File{
		{Name: "a.png", ContentType: "image/png", Data: []byte("A")},
		{Name: "b.jpg", ContentType: "image/jpeg", Data: []byte{}},
	}
	if !reflect.DeepEqual(req.Body.Images, want) {
		t.Errorf("Images = %+v, want %+v", req.Body.Images, want)
	}
}

func TestParseRequest_MultipartMissingFile(t *testing.T) {
	req, err := parseMultipart(t, multipartPart{name: "text", data: "hello"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Body.Avatar.Data != nil || req.Body.Images != nil {
		t.Errorf("expected no files, got %+v", req.Body)
	}
}

func TestParseRequest_MultipartSkipsBodyOfGetRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/chats/1/messages", nil)
	var req multipartRequest
	if err := ParseRequest(r, &req); err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
}

func TestParseRequest_MultipartErrors(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     string
	}{
		{"json content type", "application/json", `{"text":"hello"}`, "invalid multipart body"},
		{"missing boundary", "multipart/form-data", "", "invalid multipart body"},
		{"malformed part header", "multipart/form-data; boundary=b", "--b\r\nbroken header\r\n\r\n", "failed to read multipart body"},
		{
			"truncated field",
			"multipart/form-data; boundary=b",
			"--b\r\nContent-Disposition: form-data; name=\"text\"\r\n\r\nhel",
			"failed to read form field text",
		},
		{
			"text field with a wrong type",
			"multipart/form-data; boundary=b",
			"--b\r\nContent-Disposition: form-data; name=\"count\"\r\n\r\nabc\r\n--b--\r\n",
			"failed to set form field count",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/chats/1/messages", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)
			var req multipartRequest
			err := ParseRequest(r, &req)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseRequest_MultipartPartWithoutName(t *testing.T) {
	body := "--b\r\nContent-Disposition: form-data\r\n\r\nvalue\r\n--b--\r\n"
	r := httptest.NewRequest(http.MethodPost, "/chats/1/messages", strings.NewReader(body))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=b")
	var req struct {
		Body struct {
			Untagged string
			Avatar   File `gork:"avatar"`
		}
	}
	if err := ParseRequest(r, &req); err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.Body.Untagged != "" {
		t.Errorf("Untagged = %q, want empty", req.Body.Untagged)
	}
}

func TestValidateRequest_MultipartFiles(t *testing.T) {
	file := func(data string) File { return File{Name: "f", Data: []byte(data)} }
	tests := []struct {
		name    string
		avatar  File
		images  []File
		details map[string][]string
	}{
		{"valid", file("A"), []File{file("A")}, nil},
		{"empty file is present", file(""), []File{file("")}, nil},
		{"missing file", File{}, []File{file("A")}, map[string][]string{"body.avatar": {"required"}}},
		{"too few files", file("A"), nil, map[string][]string{"body.images": {"min"}}},
		{"too many files", file("A"), []File{file("A"), file("B"), file("C")}, map[string][]string{"body.images": {"max"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req multipartRequest
			req.Path.ChatID = "1"
			req.Body.Text = "hello"
			req.Body.Avatar = tt.avatar
			req.Body.Images = tt.images

			err := NewConventionValidator().ValidateRequest(context.Background(), &req)
			if tt.details == nil {
				if err != nil {
					t.Fatalf("ValidateRequest: %v", err)
				}
				return
			}
			verr, ok := err.(*ValidationErrorResponse)
			if !ok {
				t.Fatalf("error = %v, want *ValidationErrorResponse", err)
			}
			if !reflect.DeepEqual(verr.Details, tt.details) {
				t.Errorf("details = %v, want %v", verr.Details, tt.details)
			}
		})
	}
}

func TestMultipartHandler(t *testing.T) {
	var got multipartRequest
	handler := func(_ context.Context, req multipartRequest) error {
		got = req
		return nil
	}
	httpHandler, _ := createHandlerFromAny(multipartTestAdapter{}, handler)

	t.Run("accepts a valid upload", func(t *testing.T) {
		body, contentType := multipartBody(t,
			multipartPart{name: "text", data: "hello"},
			multipartPart{name: "avatar", filename: "me.png", contentType: "image/png", data: "PNG"},
			multipartPart{name: "images", filename: "a.png", contentType: "image/png", data: "A"},
		)
		r := httptest.NewRequest(http.MethodPost, "/chats/1/messages", body)
		r.SetPathValue("chat_id", "1")
		r.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		httpHandler(rec, r)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
		}
		if got.Body.Text != "hello" || string(got.Body.Avatar.Data) != "PNG" || len(got.Body.Images) != 1 {
			t.Errorf("unexpected request: %+v", got.Body)
		}
	})

	t.Run("rejects a JSON body with 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/chats/1/messages", strings.NewReader(`{"text":"hello"}`))
		r.SetPathValue("chat_id", "1")
		r.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		httpHandler(rec, r)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid multipart body") {
			t.Errorf("response = %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("reports a validation error in the Gork error format", func(t *testing.T) {
		body, contentType := multipartBody(t, multipartPart{name: "text", data: "hello"})
		r := httptest.NewRequest(http.MethodPost, "/chats/1/messages", body)
		r.SetPathValue("chat_id", "1")
		r.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		httpHandler(rec, r)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
		var resp ValidationErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(resp.Details["body.avatar"], []string{"required"}) {
			t.Errorf("details = %v", resp.Details)
		}
	})
}

func TestMultipartBodySpec(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Post("/chats/{chat_id}/messages", func(context.Context, multipartRequest) error { return nil })
	router.Post("/json", func(context.Context, sliceBodyRawRequest) error { return nil })

	spec := GenerateOpenAPI(registry)
	body := spec.Paths["/chats/{chat_id}/messages"].Post.RequestBody
	if body == nil || !body.Required || len(body.Content) != 1 {
		t.Fatalf("unexpected request body: %+v", body)
	}
	media := body.Content["multipart/form-data"]
	if media == nil || media.Schema.Ref == "" {
		t.Fatalf("expected a multipart/form-data schema reference, got %+v", body.Content)
	}
	schema := spec.Components.Schemas[strings.TrimPrefix(media.Schema.Ref, "#/components/schemas/")]

	file := &Schema{Type: "string", ContentMediaType: "application/octet-stream"}
	if schema.Type != "object" {
		t.Errorf("type = %q, want object", schema.Type)
	}
	if !reflect.DeepEqual(schema.Properties["avatar"], file) {
		t.Errorf("avatar = %+v, want %+v", schema.Properties["avatar"], file)
	}
	if want := (&Schema{Type: "array", Items: file}); !reflect.DeepEqual(schema.Properties["images"], want) {
		t.Errorf("images = %+v, want %+v", schema.Properties["images"], want)
	}
	if got := schema.Properties["text"]; got.Type != "string" || got.MaxLength == nil || *got.MaxLength != 10 {
		t.Errorf("text = %+v", got)
	}
	if got := schema.Properties["tags"]; got.Type != "array" || got.Items.Type != "string" {
		t.Errorf("tags = %+v", got)
	}
	if !reflect.DeepEqual(schema.Required, []string{"text", "avatar"}) {
		t.Errorf("required = %v", schema.Required)
	}

	data, err := json.Marshal(schema.Properties["avatar"])
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"type":"string","contentMediaType":"application/octet-stream"}`; string(data) != want {
		t.Errorf("avatar JSON = %s, want %s", data, want)
	}
}
