package stdlib

import "github.com/gork-labs/gork/pkg/api"

// DocsRoute registers routes required to serve API documentation (OpenAPI JSON
// + selected UI) under the provided path. The implementation simply delegates
// to the shared TypedRouter helper.
func (r *Router) DocsRoute(path string, cfg ...api.DocsConfig) {
	if r == nil || r.TypedRouter == nil {
		return
	}
	r.TypedRouter.DocsRoute(path, cfg...)
}
