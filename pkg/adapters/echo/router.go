package echo

import (
	"context"
	"net/http"
	"strings"

	echosdk "github.com/labstack/echo/v4"

	"github.com/gork-labs/gork/pkg/api"
)

// key used to store echo.Context in request context
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
	*api.TypedRouter[*echosdk.Echo]

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

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
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
	r.TypedRouter = &tr

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

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
		nativePath := toNativePath(newPrefix + path)
		g.Add(method, nativePath, echosdk.WrapHandler(handler))
	}

	return &Router{
		echo:       r.echo,
		group:      g,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		TypedRouter: func() *api.TypedRouter[*echosdk.Echo] {
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

func (r *Router) GetRegistry() *api.RouteRegistry { return r.registry }

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
