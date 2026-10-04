package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
)

// localHTTPClient sends each request of the validator client to a local test server.
type localHTTPClient struct {
	url string
}

func (c localHTTPClient) Post(_ string, contentType string, body io.Reader) (*http.Response, error) {
	return (&DefaultHTTPClient{}).Post(c.url, contentType, body)
}

func TestDefaultValidatorClientWithLocalServer(t *testing.T) {
	var reply string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		_, _ = io.WriteString(w, reply)
	}))
	defer server.Close()

	client := &DefaultValidatorClient{httpClient: localHTTPClient{url: server.URL}}
	spec := &api.OpenAPISpec{OpenAPI: "3.1.0", Info: api.Info{Title: "Test", Version: "1.0.0"}}

	reply = "{}"
	if err := validateSpecWithClient(spec, client); err != nil {
		t.Errorf("expected no error for a clean result, got %v", err)
	}

	reply = `{"messages":[{"level":"error","message":"bad"}]}`
	if err := validateSpecWithClient(spec, client); err == nil {
		t.Error("expected an error for a validator error message")
	}
}
