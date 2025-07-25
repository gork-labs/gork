//go:generate go run github.com/gork-labs/gork/cmd/gork openapi generate --build . --source ../.. --output ../../examples/openapi.json --title "Examples API" --version "0.1.0"
package main

import (
	"log"
	"net/http"

	"github.com/gork-labs/gork/examples"
	"github.com/gork-labs/gork/pkg/api"
)

func main() {
	mux := http.NewServeMux()

	// Register API routes.
	router := examples.RegisterRoutes(mux)

	// If this process is started for OpenAPI export (detected via GORK_EXPORT)
	// we emit the spec enriched at build-time and exit immediately so that
	// tooling such as `gork openapi generate` can capture it.
	router.ExportOpenAPIAndExit(
		api.WithTitle("Examples API"),
		api.WithVersion("0.1.0"),
	)

	// Serve API documentation at /docs (Stoplight UI by default)
	router.DocsRoute("/docs/*", api.DocsConfig{SpecFile: "examples/openapi.json"})

	for _, rt := range router.GetRegistry().GetRoutes() {
		log.Printf("registered route: %s %s", rt.Method, rt.Path)
	}

	log.Println("Server listening on :8800 (docs at http://localhost:8800/docs/)")
	log.Fatal(http.ListenAndServe(":8800", mux))
}
