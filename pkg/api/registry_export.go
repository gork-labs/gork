package api

import "encoding/json"

// ExportableRouteInfo is a JSON-serializable version of RouteInfo without
// non-marshallable fields like middleware functions.
type ExportableRouteInfo struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	HandlerName  string `json:"handlerName"`
	RequestType  string `json:"requestType,omitempty"`
	ResponseType string `json:"responseType,omitempty"`
}

// Export serialises the registered routes into JSON so that external tools can
// consume the information without importing Go types. The format is considered
// an implementation detail and MAY change between minor versions.
func (r *RouteRegistry) Export() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exportableRoutes := make([]ExportableRouteInfo, 0, len(r.routes))
	for _, route := range r.routes {
		exportable := ExportableRouteInfo{
			Method:      route.Method,
			Path:        route.Path,
			HandlerName: route.HandlerName,
		}
		if route.RequestType != nil {
			exportable.RequestType = route.RequestType.String()
		}
		if route.ResponseType != nil {
			exportable.ResponseType = route.ResponseType.String()
		}
		exportableRoutes = append(exportableRoutes, exportable)
	}

	return json.Marshal(exportableRoutes)
}
