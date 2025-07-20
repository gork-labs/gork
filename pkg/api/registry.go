package api

import (
	"reflect"
	"sync"
)

// RouteInfo contains metadata about a registered route.
// Most fields are filled by the router implementation at registration time.
// Additional fields (Method / Path) are set by the concrete router wrapper
// right before the route is added to the registry.
type RouteInfo struct {
	Method       string         // HTTP method (GET, POST, ...)
	Path         string         // Absolute route path, including any prefix
	Handler      interface{}    // The original typed handler function
	HandlerName  string         // getFunctionName(handler)
	RequestType  reflect.Type   // The concrete request struct type (non-pointer)
	ResponseType reflect.Type   // The concrete response struct type (non-pointer)
	Options      *HandlerOption // Collected handler options (tags, security, etc.)
	// Middleware can hold router specific middleware descriptors. For now we
	// simply keep them as raw Option values so that future work can refine the
	// representation without breaking the API.
	Middleware []Option
}

// RouteRegistry stores route metadata for a single router instance. It is
// intentionally not shared globally so that multiple routers can be created in
// the same process without stepping on each other's toes.
//
// The registry is safe for concurrent use by multiple goroutines.
type RouteRegistry struct {
	mu     sync.RWMutex
	routes []*RouteInfo
}

// NewRouteRegistry creates a new, empty registry.
func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{routes: make([]*RouteInfo, 0)}
}

// Register adds a route to the registry.
func (r *RouteRegistry) Register(info *RouteInfo) {
	if info == nil {
		return
	}
	r.mu.Lock()
	r.routes = append(r.routes, info)
	r.mu.Unlock()
}

// GetRoutes returns a copy of all registered routes so callers can freely
// modify the returned slice without affecting the internal state.
func (r *RouteRegistry) GetRoutes() []*RouteInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make([]*RouteInfo, len(r.routes))
	copy(cp, r.routes)
	return cp
}
