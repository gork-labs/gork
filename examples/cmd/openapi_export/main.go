package main

import (
	"encoding/json"
	"log"
	"net/http"

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
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		spec := api.GenerateOpenAPI(router.GetRegistry(), api.WithTitle("Example API"), api.WithVersion("1.0.0"))
		json.NewEncoder(w).Encode(spec)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
