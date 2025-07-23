package fiber

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/gork-labs/gork/pkg/api"
)

// GenericParameterAdapter works with any context type for maximum flexibility
type GenericParameterAdapter[T any] interface {
	Path(ctx T, key string) (string, bool)
	Query(ctx T, key string) (string, bool)
	Header(ctx T, key string) (string, bool)
	Cookie(ctx T, key string) (string, bool)
}

// FiberParameterAdapter works directly with fiber.Ctx for maximum performance
type FiberParameterAdapter = GenericParameterAdapter[*fiber.Ctx]

// HTTPParameterAdapter works with http.Request for compatibility
type HTTPParameterAdapter = GenericParameterAdapter[*http.Request]

type fiberParamAdapter struct{}

func (fiberParamAdapter) Path(c *fiber.Ctx, k string) (string, bool) {
	v := c.Params(k)
	return v, v != ""
}

func (fiberParamAdapter) Query(c *fiber.Ctx, k string) (string, bool) {
	v := c.Query(k)
	return v, v != ""
}

func (fiberParamAdapter) Header(c *fiber.Ctx, k string) (string, bool) {
	v := c.Get(k)
	return v, v != ""
}

func (fiberParamAdapter) Cookie(c *fiber.Ctx, k string) (string, bool) {
	v := c.Cookies(k)
	return v, v != ""
}

// httpRequestAdapter implements the api.ParameterAdapter interface for http.Request compatibility
type httpRequestAdapter struct {
	api.RequestParamAdapter
}

func (httpRequestAdapter) Path(r *http.Request, k string) (string, bool) {
	// For docs routes, path parameters aren't typically used
	return "", false
}

// createFiberHandler creates a Fiber handler that works directly with fiber.Ctx
// without any http.Request allocation, maximizing performance
func createFiberHandler(adapter FiberParameterAdapter, handler interface{}) (func(*fiber.Ctx) error, *api.RouteInfo) {
	v := reflect.ValueOf(handler)
	t := v.Type()

	// Validate handler signature
	if t.Kind() != reflect.Func {
		panic("handler must be a function")
	}
	if t.NumIn() != 2 {
		panic("handler must accept exactly 2 parameters (context.Context, Request)")
	}
	if !t.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		panic("first handler parameter must be context.Context")
	}
	if t.NumOut() != 2 {
		panic("handler must return (Response, error)")
	}
	if !t.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("second handler return value must be error")
	}

	reqType := t.In(1)
	respType := t.Out(0)

	// Build RouteInfo
	info := &api.RouteInfo{
		Handler:      handler,
		HandlerName:  getFunctionName(handler),
		RequestType:  reqType,
		ResponseType: respType,
		Options:      &api.HandlerOption{},
	}

	// Create Fiber handler
	fiberHandler := func(c *fiber.Ctx) error {
		// Instantiate request struct
		reqPtr := reflect.New(reqType)

		// Process request parameters directly from Fiber context
		if err := processFiberRequestParameters(reqPtr, c, adapter); err != nil {
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}

		// Validate request
		if err := validateFiberRequest(c, reqPtr.Interface()); err != nil {
			return err // Error already sent to response
		}

		// Call handler
		return processFiberHandlerResponse(c, v, reqPtr)
	}

	return fiberHandler, info
}

// createHTTPHandler creates a standard HTTP handler for compatibility (docs, etc.)
func createHTTPHandler(adapter HTTPParameterAdapter, handler http.HandlerFunc) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Create a simple adapter for regular HTTP handlers
		req := &http.Request{
			Method: c.Method(),
			URL:    &url.URL{Path: c.Path(), RawQuery: string(c.Request().URI().QueryString())},
			Header: make(http.Header),
		}
		// Copy headers
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Header.Set(string(key), string(value))
		})

		rw := &fiberResponseWriter{ctx: c}
		handler.ServeHTTP(rw, req)
		return nil
	}
}

// processFiberRequestParameters processes request parameters directly from fiber.Ctx
func processFiberRequestParameters(reqPtr reflect.Value, c *fiber.Ctx, adapter FiberParameterAdapter) error {
	// Parse path parameters
	parseFiberPathParameters(reqPtr, c, adapter)

	// Decode JSON body for methods that typically carry one
	if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
		if err := json.Unmarshal(c.Body(), reqPtr.Interface()); err != nil {
			return err
		}
	}

	// Parse query, header, cookie parameters
	parseFiberOtherParameters(reqPtr, c, adapter)

	return nil
}

// parseFiberPathParameters parses path parameters directly from fiber.Ctx
func parseFiberPathParameters(reqPtr reflect.Value, c *fiber.Ctx, adapter FiberParameterAdapter) {
	vStruct := reqPtr.Elem()
	tStruct := vStruct.Type()
	for i := 0; i < tStruct.NumField(); i++ {
		field := tStruct.Field(i)
		openapiTag := field.Tag.Get("openapi")
		if openapiTag == "" {
			continue
		}
		tagInfo := parseOpenAPITag(openapiTag)
		if tagInfo.In != "path" {
			continue
		}
		name := getFiberParameterName(tagInfo, field)
		if val, ok := adapter.Path(c, name); ok {
			setFiberFieldValue(vStruct.Field(i), field, val, []string{val})
		}
	}
}

// parseFiberOtherParameters parses query, header, and cookie parameters from fiber.Ctx
func parseFiberOtherParameters(reqPtr reflect.Value, c *fiber.Ctx, adapter FiberParameterAdapter) {
	vStruct := reqPtr.Elem()
	tStruct := vStruct.Type()
	for i := 0; i < tStruct.NumField(); i++ {
		field := tStruct.Field(i)
		openapiTag := field.Tag.Get("openapi")
		if openapiTag == "" {
			continue
		}
		tagInfo := parseOpenAPITag(openapiTag)
		name := getFiberParameterName(tagInfo, field)

		var val string
		var ok bool
		switch tagInfo.In {
		case "query":
			val, ok = adapter.Query(c, name)
		case "header":
			val, ok = adapter.Header(c, name)
		case "cookie":
			val, ok = adapter.Cookie(c, name)
		}
		if ok {
			setFiberFieldValue(vStruct.Field(i), field, val, []string{val})
		}
	}
}

// Helper functions for Fiber-specific parameter processing
func parseOpenAPITag(tag string) struct{ Name, In string } {
	parts := strings.Split(tag, ",")
	result := struct{ Name, In string }{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "in=") {
			result.In = strings.TrimPrefix(part, "in=")
		} else if result.Name == "" {
			result.Name = part
		}
	}
	return result
}

func getFiberParameterName(tagInfo struct{ Name, In string }, field reflect.StructField) string {
	if tagInfo.Name != "" {
		return tagInfo.Name
	}
	if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		return jsonTag
	}
	return field.Name
}

func setFiberFieldValue(fieldValue reflect.Value, field reflect.StructField, value string, values []string) {
	// This is a simplified version - in practice you'd want full type conversion
	if fieldValue.CanSet() && fieldValue.Kind() == reflect.String {
		fieldValue.SetString(value)
	}
}

func validateFiberRequest(c *fiber.Ctx, req interface{}) error {
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return nil
}

func processFiberHandlerResponse(c *fiber.Ctx, handlerValue reflect.Value, reqPtr reflect.Value) error {
	// Call the handler function
	results := handlerValue.Call([]reflect.Value{
		reflect.ValueOf(c.Context()),
		reqPtr.Elem(),
	})

	// Check for error
	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Send response
	response := results[0].Interface()
	return c.JSON(response)
}

func getFunctionName(handler interface{}) string {
	return reflect.TypeOf(handler).Name()
}

// fiberResponseWriter adapts fiber.Ctx to http.ResponseWriter interface for docs handlers
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

// Router wraps a Fiber app or group with TypedRouter capabilities.
// If group == nil we operate on the root Fiber app instance.
type Router struct {
	typedRouter *api.TypedRouter[*fiber.App]

	app        *fiber.App
	group      fiber.Router
	registry   *api.RouteRegistry
	prefix     string
	middleware []api.Option
}

// NewRouter creates a new router around the given Fiber app.
func NewRouter(app *fiber.App, opts ...api.Option) *Router {
	if app == nil {
		app = fiber.New()
	}

	registry := api.NewRouteRegistry()
	fiberAdapter := fiberParamAdapter{}
	httpAdapter := httpRequestAdapter{}

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
		nativePath := toNativePath(path)

		// If info is nil or info.Handler is nil, we have a regular http.HandlerFunc
		if info == nil || info.Handler == nil {
			// This is a regular HTTP handler (like docs), use HTTP adapter
			fiberHandler := createHTTPHandler(httpAdapter, handler)
			app.Add(method, nativePath, fiberHandler)
		} else {
			// This is our custom typed handler, use the Fiber-native approach
			fiberHandler, _ := createFiberHandler(fiberAdapter, info.Handler)
			app.Add(method, nativePath, fiberHandler)
		}
	}

	r := &Router{
		app:        app,
		registry:   registry,
		middleware: opts,
	}

	// Use the HTTP adapter for TypedRouter since it expects api.ParameterAdapter
	// The registerFn will handle routing to the appropriate adapter
	tr := api.NewTypedRouter[*fiber.App](
		app,
		registry,
		"",
		opts,
		httpAdapter,
		registerFn,
	)
	r.typedRouter = &tr

	return r
}

// Group creates a sub-router with prefix sharing the same registry.
func (r *Router) Group(prefix string) *Router {
	newPrefix := r.prefix + prefix
	var g fiber.Router
	if r.group != nil {
		g = r.group.Group(prefix)
	} else {
		g = r.app.Group(prefix)
	}

	fiberAdapter := fiberParamAdapter{}
	httpAdapter := httpRequestAdapter{}

	registerFn := func(method, path string, handler http.HandlerFunc, info *api.RouteInfo) {
		nativePath := toNativePath(newPrefix + path)

		// If info is nil or info.Handler is nil, we have a regular http.HandlerFunc
		if info == nil || info.Handler == nil {
			// This is a regular HTTP handler (like docs), use HTTP adapter
			fiberHandler := createHTTPHandler(httpAdapter, handler)
			g.Add(method, nativePath, fiberHandler)
		} else {
			// This is our custom typed handler, use the Fiber-native approach
			fiberHandler, _ := createFiberHandler(fiberAdapter, info.Handler)
			g.Add(method, nativePath, fiberHandler)
		}
	}

	// Use the HTTP adapter for TypedRouter compatibility
	return &Router{
		app:        r.app,
		group:      g,
		registry:   r.registry,
		prefix:     newPrefix,
		middleware: r.middleware,
		typedRouter: func() *api.TypedRouter[*fiber.App] {
			tr2 := api.NewTypedRouter[*fiber.App](
				r.app,
				r.registry,
				newPrefix,
				r.middleware,
				httpAdapter,
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
