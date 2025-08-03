package gorilla

import (
	"net/http"
	"strings"

	muxpkg "github.com/gorilla/mux"

	"github.com/gork-labs/gork/pkg/api"
)

// Router wraps gorilla/mux Router.
type Router struct {
	typedRouter *api.TypedRouter[*muxpkg.Router]

	router     *muxpkg.Router
	registry   *api.RouteRegistry
	prefix     string
	middleware []api.Option
}

type gorillaParamAdapter struct{ api.HTTPParameterAdapter }

func (gorillaParamAdapter) Path(r *http.Request, k string) (string, bool) {
	v := muxpkg.Vars(r)[k]
	return v, v != ""
}

func (gorillaParamAdapter) Query(r *http.Request, k string) (string, bool) {
	v := r.URL.Query().Get(k)
	return v, v != ""
}

func (gorillaParamAdapter) Header(r *http.Request, k string) (string, bool) {
	v := r.Header.Get(k)
	return v, v != ""
}

func (gorillaParamAdapter) Cookie(r *http.Request, k string) (string, bool) {
	if c, _ := r.Cookie(k); c != nil {
		return c.Value, true
	}
	return "", false
}

// NewRouter creates a new router around the given Gorilla mux router.
func NewRouter(r *muxpkg.Router, opts ...api.Option) *Router {
	if r == nil {
		r = muxpkg.NewRouter()
	}

	registry := api.NewRouteRegistry()

	// Placeholder registerFn, will be captured below once prefix known.
	var registerFn func(string, string, http.HandlerFunc, *api.RouteInfo)

	wrapper := &Router{
		router:     r,
		registry:   registry,
		middleware: opts,
	}

	// initial prefix ""
	registerFn = func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		r.Path(toNativePath(path)).Methods(method).Handler(handler)
	}

	tr := api.NewTypedRouter[*muxpkg.Router](
		r,
		registry,
		"",
		opts,
		gorillaParamAdapter{},
		registerFn,
	)
	wrapper.typedRouter = &tr

	return wrapper
}

// Group creates a sub-router with prefix sharing the same registry.
func (wr *Router) Group(prefix string) *Router {
	newPrefix := wr.prefix + prefix
	sub := wr.router.PathPrefix(prefix).Subrouter()

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		sub.Path(toNativePath(newPrefix + path)).Methods(method).Handler(handler)
	}

	// Create a defensive copy of middleware slice to prevent aliasing
	middlewareCopy := make([]api.Option, len(wr.middleware))
	copy(middlewareCopy, wr.middleware)

	return &Router{
		router:     sub,
		registry:   wr.registry,
		prefix:     newPrefix,
		middleware: middlewareCopy,
		typedRouter: func() *api.TypedRouter[*muxpkg.Router] {
			tr2 := api.NewTypedRouter[*muxpkg.Router](
				sub,
				wr.registry,
				newPrefix,
				middlewareCopy,
				gorillaParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

// GetRegistry returns the route registry.
func (wr *Router) GetRegistry() *api.RouteRegistry { return wr.registry }

// Get registers a GET route.
func (wr *Router) Get(path string, handler interface{}, opts ...api.Option) {
	wr.typedRouter.Register("GET", path, handler, opts...)
}

// Post registers a POST route.
func (wr *Router) Post(path string, handler interface{}, opts ...api.Option) {
	wr.typedRouter.Register("POST", path, handler, opts...)
}

// Put registers a PUT route.
func (wr *Router) Put(path string, handler interface{}, opts ...api.Option) {
	wr.typedRouter.Register("PUT", path, handler, opts...)
}

// Delete registers a DELETE route.
func (wr *Router) Delete(path string, handler interface{}, opts ...api.Option) {
	wr.typedRouter.Register("DELETE", path, handler, opts...)
}

// Patch registers a PATCH route.
func (wr *Router) Patch(path string, handler interface{}, opts ...api.Option) {
	wr.typedRouter.Register("PATCH", path, handler, opts...)
}

// Register registers a route with the given HTTP method, path and handler.
func (wr *Router) Register(method, path string, handler interface{}, opts ...api.Option) {
	wr.typedRouter.Register(method, path, handler, opts...)
}

// DocsRoute registers documentation routes.
func (wr *Router) DocsRoute(path string, cfg ...api.DocsConfig) {
	wr.typedRouter.DocsRoute(path, cfg...)
}

// ExportOpenAPIAndExit delegates to the underlying TypedRouter to export OpenAPI and exit.
func (wr *Router) ExportOpenAPIAndExit(opts ...api.OpenAPIOption) {
	wr.typedRouter.ExportOpenAPIAndExit(opts...)
}

// toNativePath converts goapi wildcard patterns ("/*") to gorilla/mux compatible
// patterns using a regex catch-all segment. Example: "/docs/*" -> "/docs/{rest:.*}".
// For all other paths it returns the input unchanged.
func toNativePath(p string) string {
	if strings.HasSuffix(p, "/*") {
		return strings.TrimSuffix(p, "/*") + "/{rest:.*}"
	}
	return p
}
