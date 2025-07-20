package api

import (
	"encoding/json"
	"log"
	"os"
)

func init() {
	if os.Getenv("GORK_EXPORT") == "1" {
		EnableOpenAPIExport()
	}
}

// exportMode guards whether the process should output the generated OpenAPI
// specification to stdout and terminate after the router is fully
// initialised. It is toggled via EnableOpenAPIExport() which is typically
// called from an init() function behind the `openapi` build tag.
var exportMode bool

// EnableOpenAPIExport activates export mode. Downstream applications are
// expected to call this very early during program start-up (e.g. inside an
// init() function) when building with `-tags openapi` so that subsequent
// route registration can be captured but the HTTP server is never started.
func EnableOpenAPIExport() {
	exportMode = true
}

// ExportOpenAPIAndExit generates an OpenAPI specification from the router's
// internal RouteRegistry, writes it to stdout as pretty-printed JSON and then
// terminates the process with exit code 0.
//
// The method is a no-op unless export mode has been previously enabled. This
// allows the same code path to be executed in normal server mode without any
// conditional compilation or additional build flags.
func (r *TypedRouter[T]) ExportOpenAPIAndExit(opts ...OpenAPIOption) {
	if !exportMode {
		return
	}

	spec := GenerateOpenAPI(r.registry, opts...)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(spec); err != nil {
		log.Fatalf("failed to encode OpenAPI spec: %v", err)
	}

	// Ensure graceful termination so that calling scripts can rely on the
	// process exiting once the specification has been emitted.
	os.Exit(0)
}
