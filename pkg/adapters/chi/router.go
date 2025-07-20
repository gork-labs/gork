package chi

import (
	"net/http"

	chibase "github.com/go-chi/chi/v5"

	"github.com/gork-labs/gork/pkg/api"
)

// path parameter adapter
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

// Router wraps a chi.Mux with TypedRouter capabilities.

type Router struct {
	*api.TypedRouter[*chibase.Mux]
	mux        *chibase.Mux
	registry   *api.RouteRegistry
	prefix     string
	middleware []api.Option
}

// NewRouter returns a new chi router wrapper. If mux is nil a new one is
// created.
func NewRouter(mux *chibase.Mux, opts ...api.Option) *Router {
	if mux == nil {
		mux = chibase.NewRouter()
	}
	registry := api.NewRouteRegistry()

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
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
	r.TypedRouter = &tr

	return r
}

// Group creates a sub-router with a path prefix that shares the same registry.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
		r.mux.Method(method, newPrefix+path, handler)
	}

	return &Router{
		mux:        r.mux,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		TypedRouter: func() *api.TypedRouter[*chibase.Mux] {
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
