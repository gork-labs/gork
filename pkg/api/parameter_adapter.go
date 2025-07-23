package api

import "net/http"

// ParameterAdapter extracts path parameters for a specific router framework.
type ParameterAdapter interface {
	Query(r *http.Request, key string) (string, bool)
	Header(r *http.Request, key string) (string, bool)
	Cookie(r *http.Request, key string) (string, bool)
	Path(r *http.Request, key string) (string, bool)
}

// RequestParamAdapter implements Query, Header, and Cookie using the standard
// *http.Request helpers. Adapters can embed this and override Path (and any
// others) as needed.
type RequestParamAdapter struct{}

// Query extracts query parameters from the HTTP request.
func (RequestParamAdapter) Query(r *http.Request, k string) (string, bool) {
	v := r.URL.Query().Get(k)
	return v, v != ""
}

// Header extracts header values from the HTTP request.
func (RequestParamAdapter) Header(r *http.Request, k string) (string, bool) {
	v := r.Header.Get(k)
	return v, v != ""
}

// Cookie extracts cookie values from the HTTP request.
func (RequestParamAdapter) Cookie(r *http.Request, k string) (string, bool) {
	if c, _ := r.Cookie(k); c != nil {
		return c.Value, true
	}
	return "", false
}

// Path returns false by default – concrete adapters must override.
func (RequestParamAdapter) Path(_ *http.Request, _ string) (string, bool) {
	panic("Path extraction not implemented for this adapter; please override RequestParamAdapter.Path")
}
