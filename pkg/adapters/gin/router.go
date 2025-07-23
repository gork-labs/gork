package gin

import (
	"context"
	"net/http"
	"strings"

	ginpkg "github.com/gin-gonic/gin"

	"github.com/gork-labs/gork/pkg/api"
)

// Key used to store *gin.Context in request context.

type ginCtxKey struct{}

// Router wraps gin Engine or RouterGroup with TypedRouter capabilities.
type Router struct {
	*api.TypedRouter[*ginpkg.Engine]

	engine     *ginpkg.Engine
	group      *ginpkg.RouterGroup
	registry   *api.RouteRegistry
	prefix     string
	middleware []api.Option
}

// NewRouter creates a new router around the given Gin engine.
func NewRouter(e *ginpkg.Engine, opts ...api.Option) *Router {
	if e == nil {
		e = ginpkg.New()
	}

	registry := api.NewRouteRegistry()

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		e.Handle(method, toNativePath(path), func(c *ginpkg.Context) {
			reqWith := c.Request.WithContext(context.WithValue(c.Request.Context(), ginCtxKey{}, c))
			handler.ServeHTTP(c.Writer, reqWith)
		})
	}

	r := &Router{
		engine:     e,
		registry:   registry,
		middleware: opts,
	}

	tr := api.NewTypedRouter[*ginpkg.Engine](
		e,
		registry,
		"",
		opts,
		ginParamAdapter{},
		registerFn,
	)
	r.TypedRouter = &tr

	return r
}

// Group creates a sub-router with prefix sharing the same registry.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix
	var g *ginpkg.RouterGroup
	if r.group != nil {
		g = r.group.Group(prefix)
	} else {
		g = r.engine.Group(prefix)
	}

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		g.Handle(method, toNativePath(newPrefix+path), ginpkg.WrapH(handler))
	}

	return &Router{
		engine:     r.engine,
		group:      g,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		TypedRouter: func() *api.TypedRouter[*ginpkg.Engine] {
			tr2 := api.NewTypedRouter[*ginpkg.Engine](
				r.engine,
				r.registry,
				newPrefix,
				r.middleware,
				ginParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

// GetRegistry returns the route registry.
func (r *Router) GetRegistry() *api.RouteRegistry { return r.registry }

func toNativePath(p string) string {
	// Convert named params {id} -> :id
	s := strings.ReplaceAll(p, "{", ":")
	s = strings.ReplaceAll(s, "}", "")

	// Convert catch-all wildcard "/*" to "/*all" so that Gin treats it as a
	// wildcard parameter. We only transform a trailing "/*" to avoid
	// unexpected replacements in the middle of the path.
	if strings.HasSuffix(s, "/*") {
		s = strings.TrimSuffix(s, "/*") + "/*all"
	}

	return s
}

type ginParamAdapter struct{ api.RequestParamAdapter }

func (ginParamAdapter) Path(r *http.Request, k string) (string, bool) {
	if gc, ok := r.Context().Value(ginCtxKey{}).(*ginpkg.Context); ok {
		v := gc.Param(k)
		return v, v != ""
	}
	return "", false
}

func (ginParamAdapter) Query(r *http.Request, k string) (string, bool) {
	v := r.URL.Query().Get(k)
	return v, v != ""
}

func (ginParamAdapter) Header(r *http.Request, k string) (string, bool) {
	v := r.Header.Get(k)
	return v, v != ""
}

func (ginParamAdapter) Cookie(r *http.Request, k string) (string, bool) {
	if c, _ := r.Cookie(k); c != nil {
		return c.Value, true
	}
	return "", false
}
