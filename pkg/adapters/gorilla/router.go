package gorilla

import (
	"net/http"
	"strings"

	muxpkg "github.com/gorilla/mux"

	"github.com/gork-labs/gork/pkg/api"
)

// Router wraps gorilla/mux Router.

type Router struct {
	*api.TypedRouter[*muxpkg.Router]

	router     *muxpkg.Router
	registry   *api.RouteRegistry
	prefix     string
	middleware []api.Option
}

type gorillaParamAdapter struct{ api.RequestParamAdapter }

func (gorillaParamAdapter) Path(r *http.Request, k string) (string, bool) {
	v := muxpkg.Vars(r)[k]
	return v, v != ""
}

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
	registerFn = func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
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
	wrapper.TypedRouter = &tr

	return wrapper
}

func (wr *Router) Group(prefix string) *Router {
	newPrefix := wr.prefix + prefix
	sub := wr.router.PathPrefix(prefix).Subrouter()

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
		sub.Path(toNativePath(newPrefix + path)).Methods(method).Handler(handler)
	}

	return &Router{
		router:     sub,
		registry:   wr.registry,
		prefix:     newPrefix,
		middleware: wr.middleware,
		TypedRouter: func() *api.TypedRouter[*muxpkg.Router] {
			tr2 := api.NewTypedRouter[*muxpkg.Router](
				sub,
				wr.registry,
				newPrefix,
				wr.middleware,
				gorillaParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

func (wr *Router) GetRegistry() *api.RouteRegistry { return wr.registry }

// toNativePath converts goapi wildcard patterns ("/*") to gorilla/mux compatible
// patterns using a regex catch-all segment. Example: "/docs/*" -> "/docs/{rest:.*}".
// For all other paths it returns the input unchanged.
func toNativePath(p string) string {
	if strings.HasSuffix(p, "/*") {
		return strings.TrimSuffix(p, "/*") + "/{rest:.*}"
	}
	return p
}
