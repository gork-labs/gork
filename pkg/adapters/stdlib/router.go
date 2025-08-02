package stdlib

import (
	"net/http"
	"strings"

	"github.com/gork-labs/gork/pkg/api"
)

// Router provides routing capabilities using Go's standard library ServeMux.
//
// Groups are emulated by keeping track of a path prefix – Go's standard library
// router does not have a native grouping facility.

type stdlibParamAdapter struct{ api.HTTPParameterAdapter }

func (stdlibParamAdapter) Path(r *http.Request, k string) (string, bool) {
	v := r.PathValue(k)
	return v, v != ""
}

func (stdlibParamAdapter) Query(r *http.Request, k string) (string, bool) {
	v := r.URL.Query().Get(k)
	return v, v != ""
}

func (stdlibParamAdapter) Header(r *http.Request, k string) (string, bool) {
	v := r.Header.Get(k)
	return v, v != ""
}

func (stdlibParamAdapter) Cookie(r *http.Request, k string) (string, bool) {
	if c, _ := r.Cookie(k); c != nil {
		return c.Value, true
	}
	return "", false
}

// Router is a wrapper around *http.ServeMux that provides higher-level routing capabilities.
type Router struct {
	typedRouter *api.TypedRouter[*http.ServeMux]
	mux         *http.ServeMux
	registry    *api.RouteRegistry
	prefix      string
	middleware  []api.Option
}

// NewRouter creates a new wrapper around the provided *http.ServeMux. If mux is
// nil, a fresh instance is allocated.
func NewRouter(mux *http.ServeMux, opts ...api.Option) *Router {
	if mux == nil {
		mux = http.NewServeMux()
	}

	registry := api.NewRouteRegistry()

	// Callback for route registration into the stdlib mux.
	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		pattern := method + " " + toNativePath(path)
		mux.HandleFunc(pattern, handler)
	}

	r := &Router{
		mux:        mux,
		registry:   registry,
		middleware: opts,
		prefix:     "", // root
	}

	tr := api.NewTypedRouter[*http.ServeMux](
		mux,
		registry,
		"", // prefix
		opts,
		stdlibParamAdapter{},
		registerFn,
	)
	r.typedRouter = &tr

	return r
}

// Group creates a sub-router that shares the same registry and path prefix.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		pattern := method + " " + toNativePath(newPrefix+path)
		r.mux.HandleFunc(pattern, handler)
	}

	return &Router{
		mux:        r.mux,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		typedRouter: func() *api.TypedRouter[*http.ServeMux] {
			tr2 := api.NewTypedRouter[*http.ServeMux](
				r.mux,
				r.registry,
				newPrefix,
				r.middleware,
				stdlibParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

// GetRegistry returns the shared registry instance.
func (r *Router) GetRegistry() *api.RouteRegistry { return r.registry }

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

// DocsRoute registers documentation routes.
func (r *Router) DocsRoute(path string, cfg ...api.DocsConfig) {
	r.typedRouter.DocsRoute(path, cfg...)
}

// ExportOpenAPIAndExit delegates to the underlying TypedRouter to export OpenAPI and exit.
func (r *Router) ExportOpenAPIAndExit(opts ...api.OpenAPIOption) {
	r.typedRouter.ExportOpenAPIAndExit(opts...)
}

// toNativePath converts the generic goapi wildcard pattern ("/*") into the
// format expected by Go's net/http ServeMux (Go 1.22+). A trailing "/*" is
// replaced with a rest-of-path capture segment "{rest...}". All other paths
// are returned unchanged because ServeMux already understands `{param}` style
// placeholders.
func toNativePath(p string) string {
	if strings.HasSuffix(p, "/*") {
		return strings.TrimSuffix(p, "/*") + "/{rest...}"
	}
	return p
}
