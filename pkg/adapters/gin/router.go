package gin

import (
	"context"
	"net/http"
	"strings"

	ginpkg "github.com/gin-gonic/gin"

	"github.com/gork-labs/gork/pkg/api"
)

// key used to store *gin.Context in request context
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

func NewRouter(e *ginpkg.Engine, opts ...api.Option) *Router {
	if e == nil {
		e = ginpkg.New()
	}

	registry := api.NewRouteRegistry()

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
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

func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix
	var g *ginpkg.RouterGroup
	if r.group != nil {
		g = r.group.Group(prefix)
	} else {
		g = r.engine.Group(prefix)
	}

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
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
