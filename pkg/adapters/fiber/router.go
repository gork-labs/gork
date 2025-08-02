package fiber

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/gork-labs/gork/pkg/api"
)

// Key used to store *fiber.Ctx in request context.
type fiberCtxKey struct{}

type fiberParamAdapter struct {
	api.HTTPParameterAdapter
}

func (fiberParamAdapter) Path(r *http.Request, k string) (string, bool) {
	// Extract fiber context from request context
	if ctx := r.Context().Value(fiberCtxKey{}); ctx != nil {
		if c, ok := ctx.(*fiber.Ctx); ok {
			v := c.Params(k)
			return v, v != ""
		}
	}
	return "", false
}

func (fiberParamAdapter) Query(r *http.Request, k string) (string, bool) {
	// Extract fiber context from request context
	if ctx := r.Context().Value(fiberCtxKey{}); ctx != nil {
		if c, ok := ctx.(*fiber.Ctx); ok {
			v := c.Query(k)
			return v, v != ""
		}
	}
	// Fallback to regular query parsing
	v := r.URL.Query().Get(k)
	return v, v != ""
}

func (fiberParamAdapter) Header(r *http.Request, k string) (string, bool) {
	// Extract fiber context from request context
	if ctx := r.Context().Value(fiberCtxKey{}); ctx != nil {
		if c, ok := ctx.(*fiber.Ctx); ok {
			v := c.Get(k)
			return v, v != ""
		}
	}
	// Fallback to regular header parsing
	v := r.Header.Get(k)
	return v, v != ""
}

func (fiberParamAdapter) Cookie(r *http.Request, k string) (string, bool) {
	// Extract fiber context from request context
	if ctx := r.Context().Value(fiberCtxKey{}); ctx != nil {
		if c, ok := ctx.(*fiber.Ctx); ok {
			v := c.Cookies(k)
			return v, v != ""
		}
	}
	// Fallback to regular cookie parsing
	if cookie, err := r.Cookie(k); err == nil {
		return cookie.Value, true
	}
	return "", false
}

// Router wraps a Fiber app with TypedRouter capabilities.
type Router struct {
	typedRouter *api.TypedRouter[*fiber.App]
	app         *fiber.App
	registry    *api.RouteRegistry
	prefix      string
	middleware  []api.Option
}

// NewRouter creates a new router around the given Fiber app.
func NewRouter(app *fiber.App, opts ...api.Option) *Router {
	if app == nil {
		app = fiber.New()
	}

	registry := api.NewRouteRegistry()

	registerFn := func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		nativePath := toNativePath(path)
		app.Add(method, nativePath, func(c *fiber.Ctx) error {
			return handleFiberRequest(c, handler)
		})
	}

	r := &Router{
		app:        app,
		registry:   registry,
		middleware: opts,
	}

	tr := api.NewTypedRouter[*fiber.App](
		app,
		registry,
		"",
		opts,
		fiberParamAdapter{},
		registerFn,
	)
	r.typedRouter = &tr

	return r
}

// HTTPRequestCreator allows dependency injection for testing.
type HTTPRequestCreator func(method, url string, body io.Reader) (*http.Request, error)

var defaultHTTPRequestCreator HTTPRequestCreator = http.NewRequest

// createHTTPRequestFromFiber creates an HTTP request from a Fiber context.
// This function is extracted to make it easily testable.
func createHTTPRequestFromFiber(c *fiber.Ctx) (*http.Request, error) {
	return createHTTPRequestFromFiberWithCreator(c, defaultHTTPRequestCreator)
}

// createHTTPRequestFromFiberWithCreator allows injecting custom request creator for testing.
func createHTTPRequestFromFiberWithCreator(c *fiber.Ctx, creator HTTPRequestCreator) (*http.Request, error) {
	req, err := creator(c.Method(), c.OriginalURL(), strings.NewReader(string(c.Body())))
	if err != nil {
		return nil, err
	}

	// Copy headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		req.Header.Add(string(key), string(value))
	})

	// Add fiber context to request context for parameter extraction
	req = req.WithContext(context.WithValue(req.Context(), fiberCtxKey{}, c))
	return req, nil
}

// handleFiberRequest processes an HTTP request through a Fiber context.
// This function is extracted to make it easily testable.
func handleFiberRequest(c *fiber.Ctx, handler http.HandlerFunc) error {
	return handleFiberRequestWithCreator(c, handler, defaultHTTPRequestCreator)
}

// handleFiberRequestWithCreator allows injecting custom request creator for testing.
func handleFiberRequestWithCreator(c *fiber.Ctx, handler http.HandlerFunc, creator HTTPRequestCreator) error {
	req, err := createHTTPRequestFromFiberWithCreator(c, creator)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create HTTP request"})
	}

	// Create response writer adapter
	rw := &fiberResponseWriter{ctx: c}
	handler.ServeHTTP(rw, req)
	return nil
}

// createRegisterFn creates a register function for a Fiber group.
// This function is extracted to make it easily testable.
func createRegisterFn(g fiber.Router, newPrefix string) func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
	return func(method, path string, handler http.HandlerFunc, _ *api.RouteInfo) {
		nativePath := toNativePath(newPrefix + path)
		g.Add(method, nativePath, func(c *fiber.Ctx) error {
			return handleFiberRequest(c, handler)
		})
	}
}

// Group creates a sub-router with prefix sharing the same registry.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix
	g := r.app.Group(prefix)
	registerFn := createRegisterFn(g, newPrefix)

	return &Router{
		app:        r.app,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		typedRouter: func() *api.TypedRouter[*fiber.App] {
			tr2 := api.NewTypedRouter[*fiber.App](
				r.app,
				r.registry,
				newPrefix,
				r.middleware,
				fiberParamAdapter{},
				registerFn,
			)
			return &tr2
		}(),
	}
}

// GetRegistry returns the route registry.
func (r *Router) GetRegistry() *api.RouteRegistry { return r.registry }

// Unwrap returns the underlying Fiber app instance.
func (r *Router) Unwrap() *fiber.App {
	return r.typedRouter.Unwrap()
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

// fiberResponseWriter implements http.ResponseWriter for Fiber compatibility.
type fiberResponseWriter struct {
	ctx *fiber.Ctx
}

func (w *fiberResponseWriter) Header() http.Header {
	h := make(http.Header)
	w.ctx.Response().Header.VisitAll(func(key, value []byte) {
		h.Add(string(key), string(value))
	})
	return h
}

func (w *fiberResponseWriter) Write(data []byte) (int, error) {
	return w.ctx.Write(data)
}

func (w *fiberResponseWriter) WriteHeader(statusCode int) {
	w.ctx.Status(statusCode)
}

// toNativePath converts {param} placeholders to :param expected by Fiber.
func toNativePath(p string) string {
	// Convert named params {id} -> :id
	s := strings.ReplaceAll(p, "{", ":")
	s = strings.ReplaceAll(s, "}", "")

	// Fiber treats trailing /* as wildcard parameter
	if strings.HasSuffix(s, "/*") {
		s = strings.TrimSuffix(s, "/*") + "/*"
	}

	return s
}
