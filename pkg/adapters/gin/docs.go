package gin

import "github.com/gork-labs/gork/pkg/api"

// DocsRoute delegates to the underlying TypedRouter implementation to expose
// documentation routes using Gin router.
func (r *Router) DocsRoute(path string, cfg ...api.DocsConfig) {
	if r == nil || r.TypedRouter == nil {
		return
	}
	r.TypedRouter.DocsRoute(path, cfg...)
}
