package stdlib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
)

type binaryRequest struct {
	Path struct {
		ID string `gork:"id"`
	}
}

type binaryResponse struct {
	Body api.Binary
}

func binaryHandler(context.Context, binaryRequest) (*binaryResponse, error) {
	return &binaryResponse{Body: api.Binary{ContentType: "image/png", Data: []byte("png")}}, nil
}

func TestBinaryRouteWithMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	NewRouter(mux).Get("/images/{id}", binaryHandler, api.WithResponseContentTypes("image/png"))
	authorized := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})

	denied := httptest.NewRecorder()
	authorized.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/images/1", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Errorf("status without a token = %d, want 401", denied.Code)
	}

	allowed := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/images/1", nil)
	req.Header.Set("Authorization", "Bearer token")
	authorized.ServeHTTP(allowed, req)
	if allowed.Code != http.StatusOK || allowed.Header().Get("Content-Type") != "image/png" || allowed.Body.String() != "png" {
		t.Errorf("response with a token = %d %q %q, want 200 image/png png", allowed.Code, allowed.Header().Get("Content-Type"), allowed.Body.String())
	}
}
