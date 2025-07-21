package gin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDocsRoute_ServesSpecAndUI ensures that the DocsRoute helper registers
// both the OpenAPI spec endpoint and the HTML UI catch-all path for Gin.
func TestDocsRoute_ServesSpecAndUI(t *testing.T) {
	r := NewRouter(nil)

	// Mount documentation under /docs/*.
	r.DocsRoute("/docs/*")

	engine := r.Unwrap()

	// --- Spec endpoint ----------------------------------------------------
	reqSpec := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	recSpec := httptest.NewRecorder()
	engine.ServeHTTP(recSpec, reqSpec)

	if recSpec.Code != http.StatusOK {
		t.Fatalf("spec endpoint status = %d, want %d", recSpec.Code, http.StatusOK)
	}

	var specBody map[string]interface{}
	if err := json.Unmarshal(recSpec.Body.Bytes(), &specBody); err != nil {
		t.Fatalf("spec endpoint returned invalid JSON: %v", err)
	}

	if v, ok := specBody["openapi"]; !ok || v != "3.1.0" {
		t.Fatalf("spec openapi version = %v, want 3.1.0", v)
	}

	// --- UI endpoint ------------------------------------------------------
	reqUI := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	recUI := httptest.NewRecorder()
	engine.ServeHTTP(recUI, reqUI)

	if recUI.Code != http.StatusOK {
		t.Fatalf("UI endpoint status = %d, want %d", recUI.Code, http.StatusOK)
	}

	ct := recUI.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("UI endpoint Content-Type = %s, want text/html", ct)
	}
}
