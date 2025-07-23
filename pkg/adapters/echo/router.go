package echo

import (
	"context"
	"net/http"
	"strings"

	echosdk "github.com/labstack/echo/v4"

	"github.com/gork-labs/gork/pkg/api"
)

// Key used to store echo.Context in request context.
type echoCtxKey struct{}

type echoParamAdapter struct{ api.RequestParamAdapter }

func (echoParamAdapter) Path(r *http.Request, k string) (string, bool) {
	if ec, ok := r.Context().Value(echoCtxKey{}).(echosdk.Context); ok {
		v := ec.Param(k)
		return v, v != ""
	}
	return "", false
}

func (echoParamAdapter) Query(r *http.Request, k string) (string, bool) {
	v := r.URL.Query().Get(k)
	return v, v != ""
}

func (echoParamAdapter) Header(r *http.Request, k string) (string, bool) {
	v := r.Header.Get(k)
	return v, v != ""
}

func (echoParamAdapter) Cookie(r *http.Request, k string) (string, bool) {
	if c, _ := r.Cookie(k); c != nil {
		return c.Value, true
	}
	return "", false
}

// Router wraps an Echo engine or group with TypedRouter capabilities.
// If group == nil we operate on the root Echo instance.
type Router struct {
	typedRouter *api.TypedRouter[*echosdk.Echo]

	echo       *echosdk.Echo
	group      *echosdk.Group
	registry   *api.RouteRegistry
	prefix     string
	middleware []api.Option
}

// NewRouter creates a new router around the given Echo instance.
func NewRouter(e *echosdk.Echo, opts ...api.Option) *Router {
	if e == nil {
		e = echosdk.New()
	}

	registry := api.NewRouteRegistry()

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		nativePath := toNativePath(path)
		e.Add(method, nativePath, func(ec echosdk.Context) error {
			// store echo.Context
			reqWith := ec.Request().WithContext(context.WithValue(ec.Request().Context(), echoCtxKey{}, ec))
			handler.ServeHTTP(ec.Response().Writer, reqWith)
			return nil
		})
	}

	r := &Router{
		echo:       e,
		registry:   registry,
		middleware: opts,
	}

	tr := api.NewTypedRouter[*echosdk.Echo](
		e,
		registry,
		"",
		opts,
		echoParamAdapter{},
		registerFn,
	)
	r.typedRouter = &tr

	return r
}

// Group creates a sub-router with prefix sharing the same registry.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix
	var g *echosdk.Group
	if r.group != nil {
		g = r.group.Group(prefix)
	} else {
		g = r.echo.Group(prefix)
	}

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		nativePath := toNativePath(newPrefix + path)
		g.Add(method, nativePath, echosdk.WrapHandler(handler))
	}

	return &Router{
		echo:       r.echo,
		group:      g,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		typedRouter: func() *api.TypedRouter[*echosdk.Echo] {
			tr2 := api.NewTypedRouter[*echosdk.Echo](
				r.echo,
				r.registry,
				newPrefix,
				r.middleware,
				echoParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

// GetRegistry returns the route registry.
func (r *Router) GetRegistry() *api.RouteRegistry { return r.registry }

// Unwrap returns the underlying Echo instance.
func (r *Router) Unwrap() *echosdk.Echo {
	return r.typedRouter.Unwrap()
}

// Get registers a GET route.
func (r *Router) Get(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Get(path, handler, opts...)
}

// Post registers a POST route.
func (r *Router) Post(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Post(path, handler, opts...)
}

// Put registers a PUT route.
func (r *Router) Put(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Put(path, handler, opts...)
}

// Delete registers a DELETE route.
func (r *Router) Delete(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Delete(path, handler, opts...)
}

// Patch registers a PATCH route.
func (r *Router) Patch(path string, handler interface{}, opts ...api.Option) {
	r.typedRouter.Patch(path, handler, opts...)
}

// DocsRoute registers documentation routes.
func (r *Router) DocsRoute(path string, cfg ...api.DocsConfig) {
	r.typedRouter.DocsRoute(path, cfg...)
}

// toNativePath converts {param} placeholders to :param expected by Echo.
func toNativePath(p string) string {
	// Convert named params {id} -> :id
	s := strings.ReplaceAll(p, "{", ":")
	s = strings.ReplaceAll(s, "}", "")

	// Echo treats trailing /* with parameter name, e.g. /*.
	if strings.HasSuffix(s, "/*") {
		s = strings.TrimSuffix(s, "/*") + "/*"
	}

	return s
}
