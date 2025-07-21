package stdlib

import (
	"net/http"
	"strings"

	"github.com/gork-labs/gork/pkg/api"
)

// Router wraps an *http.ServeMux and provides the higher-level API.Router
// capabilities defined in the design spec.
//
// Groups are emulated by keeping track of a path prefix – Go's standard library
// router does not have a native grouping facility.

type stdlibParamAdapter struct{ api.RequestParamAdapter }

func (stdlibParamAdapter) Path(r *http.Request, k string) (string, bool) {
	v := r.PathValue(k)
	return v, v != ""
}

type Router struct {
	*api.TypedRouter[*http.ServeMux]
	mux        *http.ServeMux
	registry   *api.RouteRegistry
	prefix     string
	middleware []api.Option
}

// NewRouter creates a new wrapper around the provided *http.ServeMux. If mux is
// nil, a fresh instance is allocated.
func NewRouter(mux *http.ServeMux, opts ...api.Option) *Router {
	if mux == nil {
		mux = http.NewServeMux()
	}

	registry := api.NewRouteRegistry()

	// Callback for route registration into the stdlib mux.
	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
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
	r.TypedRouter = &tr

	return r
}

// Group creates a sub-router that shares the same registry and path prefix.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
		pattern := method + " " + toNativePath(newPrefix+path)
		r.mux.HandleFunc(pattern, handler)
	}

	return &Router{
		mux:        r.mux,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		TypedRouter: func() *api.TypedRouter[*http.ServeMux] {
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
