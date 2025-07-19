package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInPathParameterGeneration(t *testing.T) {
	tempDir := t.TempDir()

	handlerCode := `package handlers
import "context"

type FooRequest struct {
    ID string ` + "`json:\"id\" openapi:\"in=path\" validate:\"required,uuid\"`" + `
    Filter string ` + "`json:\"filter\" openapi:\"in=query\"`" + `
}

func GetFoo(ctx context.Context, req FooRequest) (*struct{}, error) { return nil, nil }
`

	routesCode := `package routes
import (
    "net/http"
    "../handlers"
    "github.com/gork-labs/gork/pkg/api"
)

func Setup() {
    http.HandleFunc("GET /foo/{id}", api.HandlerFunc(handlers.GetFoo))
}
`

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "handlers.go"), []byte(handlerCode), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "routes.go"), []byte(routesCode), 0644))

	gen := New("Test API", "1.0.0")
	require.NoError(t, gen.ParseDirectories([]string{tempDir}))
	require.NoError(t, gen.ParseRoutes([]string{filepath.Join(tempDir, "routes.go")}))

	spec := gen.Generate()
	pathItem, ok := spec.Paths["/foo/{id}"]
	require.True(t, ok)
	require.NotNil(t, pathItem.Get)

	var pathParam, queryParam bool
	for _, p := range pathItem.Get.Parameters {
		if p.In == "path" && p.Name == "id" {
			pathParam = true
		}
		if p.In == "query" && p.Name == "filter" {
			queryParam = true
		}
	}
	assert.True(t, pathParam)
	assert.True(t, queryParam)
}
