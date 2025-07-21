# Simple Documentation Serving Approach

## Overview

This document presents a simpler, more immediate approach to serving API documentation that can be implemented quickly while still providing a good developer experience.

## Simplified API Design

Instead of implementing a full-featured documentation system immediately, we can start with a minimal approach:

```go
// Simple one-liner to serve docs
router.DocsRoute("/docs/*")

// With basic customization
router.DocsRoute("/api-docs/*", api.DocsConfig{
    Title: "My API Documentation",
})

// With CORS configuration (using router's native middleware)
router.DocsRoute("/docs/*", api.DocsConfig{
    Title: "My API Documentation",
    // CORS will be handled by router's middleware
})
```

## Minimal Implementation

### Step 1: Add DocsRoute to TypedRouter

```go
// pkg/api/typed_router.go

// DocsConfig holds basic documentation configuration
type DocsConfig struct {
    Title       string
    OpenAPIPath string // defaults to "/openapi.json"
    UI          string // "stoplight", "swagger", "redoc"
}

// DocsRoute registers a documentation UI route
func (r *TypedRouter[T]) DocsRoute(path string, config ...DocsConfig) {
    cfg := DocsConfig{
        Title:       "API Documentation",
        OpenAPIPath: "/openapi.json",
        UI:          "stoplight",
    }
    
    if len(config) > 0 {
        cfg = config[0]
    }
    
    // Ensure path ends with /*
    basePath := strings.TrimSuffix(path, "/*")
    
    // Register OpenAPI endpoint
    r.Get(cfg.OpenAPIPath, func(ctx context.Context, _ *struct{}) (*OpenAPIResponse, error) {
        spec := r.registry.GenerateOpenAPI()
        return &OpenAPIResponse{Body: spec}, nil
    })
    
    // Register docs UI
    r.registerFn("GET", basePath + "/*", r.createDocsHandler(basePath, cfg), nil)
}

type OpenAPIResponse struct {
    Body interface{} `json:"-" contentType:"application/json"`
}
```

### Step 2: Create Simple Docs Handler

```go
// pkg/api/docs_handler.go

func (r *TypedRouter[T]) createDocsHandler(basePath string, config DocsConfig) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        // Determine which UI to serve
        var htmlTemplate string
        
        switch config.UI {
        case "swagger":
            htmlTemplate = swaggerUITemplate
        case "redoc":
            htmlTemplate = redocTemplate
        default:
            htmlTemplate = stoplightTemplate
        }
        
        // Inject configuration
        html := strings.ReplaceAll(htmlTemplate, "{{.Title}}", config.Title)
        html = strings.ReplaceAll(html, "{{.OpenAPIPath}}", config.OpenAPIPath)
        html = strings.ReplaceAll(html, "{{.BasePath}}", basePath)
        
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Write([]byte(html))
    }
}

// CDN-based templates
const stoplightTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css">
</head>
<body>
    <elements-api
        apiDescriptionUrl="{{.OpenAPIPath}}"
        router="hash"
        layout="sidebar"
    />
</body>
</html>`

const swaggerUITemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "{{.OpenAPIPath}}",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.SwaggerUIStandalonePreset
                ],
                layout: "BaseLayout"
            });
        }
    </script>
</body>
</html>`

const redocTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { margin: 0; padding: 0; }
    </style>
</head>
<body>
    <redoc spec-url="{{.OpenAPIPath}}"></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`
```

### Step 3: Update Adapters

For each adapter, we just need to ensure they properly delegate to TypedRouter:

```go
// Example for stdlib adapter
// pkg/adapters/stdlib/router.go

// DocsRoute serves API documentation UI
func (r *Router) DocsRoute(path string, config ...api.DocsConfig) {
    r.api.DocsRoute(path, config...)
}
```

## Usage Examples

### Basic Usage with http.ServeMux

```go
package main

import (
    "net/http"
    "github.com/yourusername/goapi/pkg/api"
    "github.com/yourusername/goapi/pkg/adapters/stdlib"
)

func main() {
    // Create registry and router
    registry := api.NewRouteRegistry()
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux, registry)
    
    // Register your API routes
    router.Get("/users", getUsers)
    router.Post("/users", createUser)
    
    // Serve documentation at /docs
    router.DocsRoute("/docs/*")
    
    http.ListenAndServe(":8080", mux)
}
```

### With CORS Using Router Middleware

Different routers provide different CORS solutions. Here are examples:

#### Standard Library (using a third-party middleware)

```go
import (
    "github.com/rs/cors"
)

func main() {
    registry := api.NewRouteRegistry()
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux, registry)
    
    // Register routes
    router.Get("/users", getUsers)
    router.DocsRoute("/docs/*")
    
    // Apply CORS middleware
    c := cors.New(cors.Options{
        AllowedOrigins: []string{"http://localhost:3000"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
        AllowedHeaders: []string{"Content-Type", "Authorization"},
        AllowCredentials: true,
    })
    
    handler := c.Handler(mux)
    http.ListenAndServe(":8080", handler)
}
```

#### Chi Router (native middleware)

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
)

func main() {
    registry := api.NewRouteRegistry()
    r := chi.NewRouter()
    
    // Chi's CORS middleware
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{"https://app.example.com"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
        AllowCredentials: true,
        MaxAge:           300,
    }))
    
    router := chiAdapter.NewRouter(r, registry)
    router.DocsRoute("/docs/*")
    
    http.ListenAndServe(":8080", r)
}
```

#### Echo (native middleware)

```go
import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    registry := api.NewRouteRegistry()
    e := echo.New()
    
    // Echo's CORS middleware
    e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins: []string{"http://localhost:3000"},
        AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
    }))
    
    router := echoAdapter.NewRouter(e, registry)
    router.DocsRoute("/docs/*")
    
    e.Start(":8080")
}
```

#### Gin (native middleware)

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
)

func main() {
    registry := api.NewRouteRegistry()
    g := gin.Default()
    
    // Gin's CORS middleware
    g.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        AllowCredentials: true,
    }))
    
    router := ginAdapter.NewRouter(g, registry)
    router.DocsRoute("/docs/*")
    
    g.Run(":8080")
}
```

### Multiple Documentation Endpoints

```go
func main() {
    registry := api.NewRouteRegistry()
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux, registry)
    
    // Register API routes
    router.Get("/api/v1/users", getUsers)
    router.Post("/api/v1/users", createUser)
    
    // Different documentation UIs
    router.DocsRoute("/docs/*")  // Default Stoplight
    router.DocsRoute("/swagger/*", api.DocsConfig{
        Title: "API Documentation - Swagger UI",
        UI:    "swagger",
    })
    router.DocsRoute("/redoc/*", api.DocsConfig{
        Title: "API Documentation - Redoc",
        UI:    "redoc",
    })
    
    http.ListenAndServe(":8080", mux)
}
```

## Advantages of This Approach

1. **Minimal Code**: Only ~100 lines of code to implement
2. **Uses CDN**: No asset embedding complexity
3. **Fast Implementation**: Can be done in 1-2 days
4. **Router-Native CORS**: Leverages existing middleware solutions
5. **Zero Dependencies**: Just HTML templates

## Future Enhancements

Once the basic version is working, we can incrementally add:

1. **Custom Themes**: Via configuration options
2. **Authentication**: Using router middleware
3. **Advanced Features**: Custom CSS/JS injection
4. **OpenAPI Versioning**: Support for 3.0.x and 3.1.x

## Implementation Checklist

- [ ] Add `DocsConfig` struct to `pkg/api/types.go`
- [ ] Add `DocsRoute` method to `TypedRouter`
- [ ] Create `docs_handler.go` with HTML templates
- [ ] Update each adapter to expose `DocsRoute`
- [ ] Add basic tests
- [ ] Update examples
- [ ] Document in README

## Estimated Timeline

- Implementation: 1-2 days
- Testing: 1 day
- Documentation: 0.5 days
- **Total: 2.5-3.5 days**

This approach provides immediate value while keeping the implementation simple and leveraging existing router middleware for cross-cutting concerns like CORS. 