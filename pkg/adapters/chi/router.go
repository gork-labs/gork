package chi

import (
	"net/http"

	chibase "github.com/go-chi/chi/v5"

	"github.com/gork-labs/gork/pkg/api"
)

// path parameter adapter.
type chiParamAdapter struct{ api.RequestParamAdapter }

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

// Get delegates to the TypedRouter's Get method.
func (r *Router) Get(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Get(path, handler, opts...)
}

// Post delegates to the TypedRouter's Post method.
func (r *Router) Post(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Post(path, handler, opts...)
}

// Put delegates to the TypedRouter's Put method.
func (r *Router) Put(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Put(path, handler, opts...)
}

// Delete delegates to the TypedRouter's Delete method.
func (r *Router) Delete(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Delete(path, handler, opts...)
}

// Patch delegates to the TypedRouter's Patch method.
func (r *Router) Patch(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Patch(path, handler, opts...)
}

// DocsRoute delegates to the TypedRouter's DocsRoute method.
func (r *Router) DocsRoute(path string, cfg ...api.DocsConfig) {
	r.typedRouter.DocsRoute(path, cfg...)
}
