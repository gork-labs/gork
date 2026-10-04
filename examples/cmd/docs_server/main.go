//go:generate go run github.com/gork-labs/gork/cmd/gork openapi generate --build . --source ../.. --output ../../examples/openapi.json --title "Examples API" --version "0.1.0"
package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gork-labs/gork/examples"
	"github.com/gork-labs/gork/pkg/api"
	"github.com/gork-labs/gork/pkg/rules"
)

// contextVariablesMiddleware sets up context variables for rule evaluation.
// In a real application, this would extract user info from JWT tokens, sessions, etc.
func contextVariablesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For demo purposes, set a fixed current_user
		// In production, extract this from JWT token, session, etc.
		ctx := rules.WithContextVars(r.Context(), rules.ContextVars{
			"current_user": "alice", // Set the current user for rule evaluation
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	mux := http.NewServeMux()

	// Register API routes.
	router := examples.RegisterRoutes(mux)

	// Export OpenAPI spec and exit if this is a CLI generation run
	// The CLI tool will set GORK_EXPORT=1 when it needs the spec
	if os.Getenv("GORK_EXPORT") == "1" {
		spec := api.GenerateOpenAPI(router.GetRegistry(), api.WithTitle("Examples API"), api.WithVersion("0.1.0"))
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(spec)
		return
	}

	// Serve API documentation at /docs (Stoplight UI by default)
	router.DocsRoute("/docs/*", api.DocsConfig{SpecFile: "examples/openapi.json"})

	for _, rt := range router.GetRegistry().GetRoutes() {
		log.Printf("registered route: %s %s", rt.Method, rt.Path)
	}

	server := &http.Server{
		Addr:         ":8800",
		Handler:      contextVariablesMiddleware(mux), // Add middleware to set context variables
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return context.Background()
		},
	}

	log.Println("Server listening on :8800 (docs at http://localhost:8800/docs/)")
	log.Fatal(server.ListenAndServe())
}
