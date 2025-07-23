// Package main provides an example of exporting OpenAPI specifications.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gork-labs/gork/examples"
	"github.com/gork-labs/gork/pkg/api"
)

func main() {
	mux := http.NewServeMux()

	// Register routes and capture the router instance
	router := examples.RegisterRoutes(mux)

	// Export and exit when GORK_EXPORT=1 (handled in api package)
	router.ExportOpenAPIAndExit(
		api.WithTitle("Example API"),
		api.WithVersion("1.0.0"),
	)

	// Serve spec for manual inspection
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		spec := api.GenerateOpenAPI(router.GetRegistry(), api.WithTitle("Example API"), api.WithVersion("1.0.0"))
		if err := json.NewEncoder(w).Encode(spec); err != nil {
			log.Printf("failed to encode spec: %v", err)
		}
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return context.Background()
		},
	}

	log.Println("listening on :8080")
	log.Fatal(server.ListenAndServe())
}
