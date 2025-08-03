package api

import (
	"net/http"
)

// TypedRouter provides strongly-typed methods for route registration while
// delegating the actual path handling to a framework-specific callback provided
// by the adapter wrapper.
//
// The generic parameter `T` represents the concrete underlying router type
// (e.g. *http.ServeMux, *echo.Echo, …). Keeping it generic allows callers to
// access the underlying router without type assertions via the `Unwrap()`
// helper.
type TypedRouter[T any] struct {
	underlying T
	registry   *RouteRegistry
	adapter    GenericParameterAdapter[*http.Request]
	prefix     string
	middleware []Option
	registerFn func(method, path string, handler http.HandlerFunc, info *RouteInfo)
}

// GetRegistry satisfies the Router contract for wrappers that embed
// TypedRouter.
func (r *TypedRouter[T]) GetRegistry() *RouteRegistry {
	return r.registry
}

// Unwrap returns the underlying router value.
func (r *TypedRouter[T]) Unwrap() T {
	return r.underlying
}

// NewTypedRouter is a small helper that allocates a TypedRouter value with the
// provided configuration. It is exported so that adapter packages residing in
// sub-packages of api (e.g. adapters/stdlib) can create initialised instances
// without relying on internal field access.
func NewTypedRouter[T any](underlying T, registry *RouteRegistry, prefix string, middleware []Option, adapter GenericParameterAdapter[*http.Request], registerFn func(method, path string, handler http.HandlerFunc, info *RouteInfo)) TypedRouter[T] {
	return TypedRouter[T]{
		underlying: underlying,
		registry:   registry,
		prefix:     prefix,
		middleware: middleware,
		adapter:    adapter,
		registerFn: registerFn,
	}
}

// CopyMiddleware returns a shallow copy of the middleware slice so that router
// implementations can propagate it to sub-routers when creating groups.
func (r *TypedRouter[T]) CopyMiddleware() []Option {
	cp := make([]Option, len(r.middleware))
	copy(cp, r.middleware)
	return cp
}

// --- Route registration ----------------------------------------------------

// Register registers a route with the given HTTP method, path and handler.
// The method parameter should be a standard HTTP method like "GET", "POST",
// "PUT", "DELETE", "PATCH", etc.
//
// The handler parameter should be a function with the signature:
// func(context.Context, RequestType) (ResponseType, error)
//
// Note: Until Go supports method-level type parameters on non-generic
// receivers in a stable release, we expose untyped registration helpers. These
// still provide compile-time safety because callers must pass a function that
// matches the expected signature. We perform a runtime check to be safe.
func (r *TypedRouter[T]) Register(method, path string, handler interface{}, opts ...Option) {
	// We expect the handler to be a func(context.Context, Req) (Resp, error).
	// Since we cannot express this generically at compile time, we rely on the
	// helper below to reflect on the function and validate its shape. If the
	// check fails we panic so that issues surface during development.

	allOpts := append([]Option{}, r.middleware...)
	allOpts = append(allOpts, opts...)
	httpHandler, info := createHandlerFromAny(r.adapter, handler, allOpts...)

	// Validate that Body sections are not used with read-only HTTP methods
	validateBodyUsageForMethod(method, info.RequestType)

	// Fill remaining route information.
	info.Method = method
	info.Path = r.prefix + path

	// Register metadata first so that generators can discover the route even
	// if the underlying router delays internal registration.
	r.registry.Register(info)

	if r.registerFn != nil {
		r.registerFn(method, path, httpHandler, info)
	}
}

// Get registers a GET route with the given path and handler.
func (r *TypedRouter[T]) Get(path string, handler interface{}, opts ...Option) {
	r.Register("GET", path, handler, opts...)
}

// Post registers a POST route with the given path and handler.
func (r *TypedRouter[T]) Post(path string, handler interface{}, opts ...Option) {
	r.Register("POST", path, handler, opts...)
}

// Put registers a PUT route with the given path and handler.
func (r *TypedRouter[T]) Put(path string, handler interface{}, opts ...Option) {
	r.Register("PUT", path, handler, opts...)
}

// Delete registers a DELETE route with the given path and handler.
func (r *TypedRouter[T]) Delete(path string, handler interface{}, opts ...Option) {
	r.Register("DELETE", path, handler, opts...)
}

// Patch registers a PATCH route with the given path and handler.
func (r *TypedRouter[T]) Patch(path string, handler interface{}, opts ...Option) {
	r.Register("PATCH", path, handler, opts...)
}
