package api

import "net/http"

// GenericParameterAdapter works with any context type for maximum flexibility.
// This allows framework-specific adapters to work directly with their native
// context types (e.g., *fiber.Ctx, *gin.Context) without HTTP request conversion.
type GenericParameterAdapter[T any] interface {
	Path(ctx T, key string) (string, bool)
	Query(ctx T, key string) (string, bool)
	Header(ctx T, key string) (string, bool)
	Cookie(ctx T, key string) (string, bool)
}

// HTTPParameterAdapter implements Query, Header, and Cookie using the standard
// *http.Request helpers. Adapters can embed this and override Path (and any
// others) as needed.
type HTTPParameterAdapter struct{}

// Query extracts query parameters from the HTTP request.
func (HTTPParameterAdapter) Query(r *http.Request, k string) (string, bool) {
	v := r.URL.Query().Get(k)
	return v, v != ""
}

// Header extracts header values from the HTTP request.
func (HTTPParameterAdapter) Header(r *http.Request, k string) (string, bool) {
	v := r.Header.Get(k)
	return v, v != ""
}

// Cookie extracts cookie values from the HTTP request.
func (HTTPParameterAdapter) Cookie(r *http.Request, k string) (string, bool) {
	if c, _ := r.Cookie(k); c != nil {
		return c.Value, true
	}
	return "", false
}

// Path returns false by default – concrete adapters must override.
func (HTTPParameterAdapter) Path(_ *http.Request, _ string) (string, bool) {
	panic("Path extraction not implemented for this adapter; please override HTTPParameterAdapter.Path")
}
