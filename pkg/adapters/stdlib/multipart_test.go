package stdlib

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
)

type uploadTestRequest struct {
	Path struct {
		ChatID string `gork:"chat_id" validate:"required"`
	}
	Body struct {
		Text   string     `gork:"text" validate:"required"`
		Images []api.File `gork:"images" validate:"max=1"`
	}
}

func requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TestMultipartRouteRunsRouteMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	var got uploadTestRequest
	NewRouter(mux).Post("/chats/{chat_id}/messages", func(_ context.Context, req uploadTestRequest) error {
		got = req
		return nil
	})
	server := requireToken(mux)

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	_ = form.WriteField("text", "hello")
	file, _ := form.CreateFormFile("images", "a.png")
	_, _ = file.Write([]byte("PNG"))
	_ = form.Close()

	send := func(token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/chats/7/messages", bytes.NewReader(body.Bytes()))
		r.Header.Set("Content-Type", form.FormDataContentType())
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, r)
		return rec
	}

	if rec := send(""); rec.Code != http.StatusUnauthorized || got.Body.Text != "" {
		t.Fatalf("without a token: status = %d, handler ran = %v", rec.Code, got.Body.Text != "")
	}

	if rec := send("secret"); rec.Code != http.StatusNoContent {
		t.Fatalf("with a token: status = %d, body %s", rec.Code, rec.Body.String())
	}
	if got.Path.ChatID != "7" || got.Body.Text != "hello" || len(got.Body.Images) != 1 || got.Body.Images[0].Name != "a.png" || string(got.Body.Images[0].Data) != "PNG" {
		t.Errorf("unexpected request: %+v", got)
	}
}
