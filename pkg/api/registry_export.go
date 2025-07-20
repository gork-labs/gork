package api

import "encoding/json"

// Export serialises the registered routes into JSON so that external tools can
// consume the information without importing Go types. The format is considered
// an implementation detail and MAY change between minor versions.
func (r *RouteRegistry) Export() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return json.Marshal(r.routes)
}
