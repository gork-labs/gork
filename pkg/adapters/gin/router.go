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
	typedRouter *api.TypedRouter[*ginpkg.Engine]

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
	r.typedRouter = &tr

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

	// Create a defensive copy of middleware slice to prevent aliasing
	middlewareCopy := make([]api.Option, len(r.middleware))
	copy(middlewareCopy, r.middleware)

	return &Router{
		engine:     r.engine,
		group:      g,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: middlewareCopy,
		typedRouter: func() *api.TypedRouter[*ginpkg.Engine] {
			tr2 := api.NewTypedRouter[*ginpkg.Engine](
				r.engine,
				r.registry,
				newPrefix,
				middlewareCopy,
				ginParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

// GetRegistry returns the route registry.
func (r *Router) GetRegistry() *api.RouteRegistry { return r.registry }

// Unwrap returns the underlying gin.Engine instance.
func (r *Router) Unwrap() *ginpkg.Engine {
	return r.engine
}

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

type ginParamAdapter struct{ api.HTTPParameterAdapter }

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
