package chi

import (
	"net/http"

	chibase "github.com/go-chi/chi/v5"

	"github.com/gork-labs/gork/pkg/api"
)

// path parameter adapter.
type chiParamAdapter struct{ api.HTTPParameterAdapter }

func (chiParamAdapter) Path(r *http.Request, k string) (string, bool) {
	v := chibase.URLParamFromCtx(r.Context(), k)
	return v, v != ""
}

func (chiParamAdapter) Query(r *http.Request, k string) (string, bool) {
	v := r.URL.Query().Get(k)
	return v, v != ""
}

func (chiParamAdapter) Header(r *http.Request, k string) (string, bool) {
	v := r.Header.Get(k)
	return v, v != ""
}

func (chiParamAdapter) Cookie(r *http.Request, k string) (string, bool) {
	if c, _ := r.Cookie(k); c != nil {
		return c.Value, true
	}
	return "", false
}

// Router wraps a chi.Mux with TypedRouter capabilities, enabling integration
// with Gork's API framework for route registration and middleware handling.
type Router struct {
	typedRouter *api.TypedRouter[*chibase.Mux]
	mux         *chibase.Mux
	registry    *api.RouteRegistry
	prefix      string
	middleware  []api.Option
}

// NewRouter returns a new chi router wrapper. If mux is nil a new one is
// created.
func NewRouter(mux *chibase.Mux, opts ...api.Option) *Router {
	if mux == nil {
		mux = chibase.NewRouter()
	}
	registry := api.NewRouteRegistry()

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		mux.Method(method, path, handler)
	}

	r := &Router{
		mux:        mux,
		registry:   registry,
		middleware: opts,
	}

	tr := api.NewTypedRouter[*chibase.Mux](
		mux,
		registry,
		"",
		opts,
		chiParamAdapter{},
		registerFn,
	)
	r.typedRouter = &tr

	return r
}

// Group creates a sub-router with a path prefix that shares the same registry.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		r.mux.Method(method, newPrefix+path, handler)
	}

	return &Router{
		mux:        r.mux,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		typedRouter: func() *api.TypedRouter[*chibase.Mux] {
			tr2 := api.NewTypedRouter[*chibase.Mux](
				r.mux,
				r.registry,
				newPrefix,
				r.middleware,
				chiParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

// GetRegistry exposes the shared registry instance.
func (r *Router) GetRegistry() *api.RouteRegistry { return r.registry }

// Unwrap returns the underlying chi.Mux instance.
func (r *Router) Unwrap() *chibase.Mux {
	return r.mux
}

// Get registers a GET route.
func (r *Router) Get(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Register("GET", path, handler, opts...)
}

// Post registers a POST route.
func (r *Router) Post(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Register("POST", path, handler, opts...)
}

// Put registers a PUT route.
func (r *Router) Put(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Register("PUT", path, handler, opts...)
}

// Delete registers a DELETE route.
func (r *Router) Delete(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Register("DELETE", path, handler, opts...)
}

// Patch registers a PATCH route.
func (r *Router) Patch(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Register("PATCH", path, handler, opts...)
}

// Register registers a route with the given HTTP method, path and handler.
func (r *Router) Register(method, path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Register(method, path, handler, opts...)
}

// DocsRoute delegates to the TypedRouter's DocsRoute method.
func (r *Router) DocsRoute(path string, cfg ...api.DocsConfig) {
	r.typedRouter.DocsRoute(path, cfg...)
}

// ExportOpenAPIAndExit delegates to the underlying TypedRouter to export OpenAPI and exit.
func (r *Router) ExportOpenAPIAndExit(opts ...api.OpenAPIOption) {
	r.typedRouter.ExportOpenAPIAndExit(opts...)
}
