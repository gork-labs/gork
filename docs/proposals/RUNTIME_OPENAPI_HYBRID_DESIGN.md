# Hybrid OpenAPI Generation Design: Runtime Registration + AST Documentation

## Executive Summary

This design combines runtime route registration with focused AST parsing for documentation extraction only. Security options, tags, and other metadata are captured at runtime through the existing option system, while AST is used solely for extracting Go doc comments.

## Core Principles

1. **Runtime for Routes & Options**: Use runtime registration to discover routes and their options
2. **AST for Documentation Only**: Extract comments from structs, fields, and handlers
3. **No Magic Comments**: Use standard Go doc comments
4. **No Redundant Parsing**: Don't parse what's already available at runtime

## Architecture Overview

### 1. Runtime Route Registration (Current State)

Routes self-register at runtime with complete metadata including options:

```go
func (r *TypedRouter) Post(path string, handler interface{}, opts ...Option) {
    // Extract handler metadata
    info := &RouteInfo{
        Method:       "POST",
        Path:         path,
        Handler:      handler,
        HandlerName:  getFunctionName(handler),
        RequestType:  extractRequestType(handler),
        ResponseType: extractResponseType(handler),
        Options:      mergeOptions(opts...), // This already captures WithBasicAuth, WithTags, etc.
    }
    
    r.registry.Register(info)
    r.router.Post(path, wrapHandler(handler), opts...)
}
```

The `HandlerOption` already captures:
- Security requirements (from `WithBasicAuth()`, `WithBearerTokenAuth()`, etc.)
- Tags (from `WithTags()`)
- Any other metadata passed through options

### 2. AST Documentation Extractor (Simplified)

Extract only documentation comments:

```go
// pkg/api/openapi/doc_extractor.go
type DocExtractor struct {
    packages map[string]*ast.Package  // package path -> AST
    docs     map[string]Documentation // type/function name -> docs
}

type Documentation struct {
    Description string
    Fields      map[string]FieldDoc  // for structs
    Deprecated  bool                 // from "Deprecated:" in comments
    Example     string               // from "Example:" blocks
    Since       string               // from "Since:" annotations
}

type FieldDoc struct {
    Description string
    Example     string
    Deprecated  bool
}

// Extract documentation for a type
func (e *DocExtractor) ExtractTypeDoc(typeName string) Documentation {
    // Find type declaration and extract its doc comment
    // Parse standard Go doc conventions:
    // - First paragraph is summary
    // - "Deprecated:" prefix marks deprecation
    // - "Example:" blocks contain examples
    // No magic comments or special syntax needed
}
```

### 3. Hybrid OpenAPI Generator

Combines runtime data with AST documentation:

```go
// pkg/api/openapi/generator.go
type HybridGenerator struct {
    registry     *RouteRegistry
    docExtractor *DocExtractor
}

func (g *HybridGenerator) Generate(opts ...GeneratorOption) *OpenAPISpec {
    spec := &OpenAPISpec{
        OpenAPI: "3.1.0",
        Info: Info{
            Title:   "API",
            Version: "1.0.0",
        },
    }
    
    // Process each registered route
    for _, route := range g.registry.GetRoutes() {
        // Build operation from runtime data
        operation := g.buildOperation(route)
        
        // Enhance with AST-extracted documentation only
        g.enhanceWithDocs(operation, route)
        
        // Security, tags, etc. come from route.Options (runtime data)
        if route.Options != nil {
            operation.Tags = route.Options.Tags
            operation.Security = g.convertSecurity(route.Options.Security)
            if route.Options.Deprecated {
                operation.Deprecated = true
            }
        }
        
        addOperationToSpec(spec, route.Method, route.Path, operation)
    }
    
    return spec
}

func (g *HybridGenerator) enhanceWithDocs(op *Operation, route *RouteInfo) {
    // Get handler documentation
    handlerDoc := g.docExtractor.ExtractFunctionDoc(route.HandlerName)
    if handlerDoc.Description != "" {
        op.Description = handlerDoc.Description
    }
    
    // Get request type documentation
    if route.RequestType != nil {
        reqDoc := g.docExtractor.ExtractTypeDoc(route.RequestType.Name())
        g.enhanceSchemaWithDocs(op.RequestBody.Content["application/json"].Schema, reqDoc)
    }
    
    // Get response type documentation
    if route.ResponseType != nil {
        respDoc := g.docExtractor.ExtractTypeDoc(route.ResponseType.Name())
        g.enhanceSchemaWithDocs(op.Responses["200"].Content["application/json"].Schema, respDoc)
    }
}
```

## Implementation Strategy

### Phase 1: CLI Tool (`gork`)

```go
// cmd/gork/main.go
package main

import (
    "github.com/spf13/cobra"
    "github.com/gork-labs/gork/internal/cli/openapi"
)

var rootCmd = &cobra.Command{
    Use:   "gork",
    Short: "Gork development tools",
}

func init() {
    rootCmd.AddCommand(openapi.NewCommand())
}

// cmd/gork/openapi/generate.go
var generateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate OpenAPI specification",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Step 1: Build with openapi tag
        binary, err := buildWithOpenAPITag(buildPath)
        if err != nil {
            return fmt.Errorf("build failed: %w", err)
        }
        defer os.Remove(binary)
        
        // Step 2: Extract runtime routes
        registry, err := extractRegistry(binary)
        if err != nil {
            return fmt.Errorf("failed to extract routes: %w", err)
        }
        
        // Step 3: Parse source for documentation
        docExtractor := openapi.NewDocExtractor()
        if err := docExtractor.ParseDirectory(sourcePath); err != nil {
            return fmt.Errorf("failed to parse source: %w", err)
        }
        
        // Step 4: Generate combined spec
        generator := openapi.NewHybridGenerator(registry, docExtractor)
        spec := generator.Generate(
            openapi.WithTitle(title),
            openapi.WithVersion(version),
        )
        
        // Step 5: Write output
        return writeSpec(spec, outputPath, format)
    },
}
```

### Phase 2: Registry Export Enhancement

Ensure the runtime registry export includes all necessary information:

```go
// RouteInfo already contains Options which has Security, Tags, etc.
type RouteInfo struct {
    Method       string
    Path         string
    Handler      interface{}
    HandlerName  string
    RequestType  reflect.Type
    ResponseType reflect.Type
    Options      *HandlerOption  // This has everything from WithBasicAuth, WithTags, etc.
    Middleware   []Option
}

// Add JSON marshaling support for registry export
func (r *RouteRegistry) Export() ([]byte, error) {
    routes := r.GetRoutes()
    // Convert to JSON-serializable format
    return json.Marshal(routes)
}
```

## Registry Export Helper

Instead of manual environment checks, provide a unified helper:

```go
// pkg/api/export.go
package api

// EnableOpenAPIExport enables OpenAPI schema export mode.
// Call this in your main() before setting up routes.
func EnableOpenAPIExport() {
    exportMode = true
}

// ExportOpenAPIAndExit exports the OpenAPI schema and exits if in export mode.
// Call this after all routes are registered.
func (r *TypedRouter) ExportOpenAPIAndExit(opts ...OpenAPIOption) {
    if !exportMode {
        return
    }
    
    spec := GenerateOpenAPI(r.registry, opts...)
    if err := json.NewEncoder(os.Stdout).Encode(spec); err != nil {
        log.Fatal(err)
    }
    os.Exit(0)
}
```

Usage in applications:

```go
//go:build openapi

package main

import "github.com/gork-labs/gork/pkg/api"

func init() {
    api.EnableOpenAPIExport()
}
```

```go
// main.go
func main() {
    router := stdlib.NewRouter(http.NewServeMux())
    
    // Register all routes
    setupRoutes(router)
    
    // Export and exit if in openapi mode
    router.ExportOpenAPIAndExit(
        api.WithTitle("My API"),
        api.WithVersion("1.0.0"),
    )
    
    // Normal server startup
    http.ListenAndServe(":8080", router)
}
```

## Example: Complete Flow

Given this code:

```go
// CreateUser creates a new user account.
// This endpoint validates the input and ensures the username is unique.
// Passwords are hashed using bcrypt before storage.
//
// Deprecated: Use CreateUserV2 for better validation
func CreateUser(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
    // Implementation
}

// CreateUserRequest represents the request to create a user.
type CreateUserRequest struct {
    // Username must be unique across the system
    Username string `json:"username" validate:"required,min=3,max=20"`
    
    // Email address for account recovery
    Email string `json:"email" validate:"required,email"`
}

// In routes.go:
router.Post("/users", CreateUser, 
    api.WithBasicAuth(),      // Captured at runtime
    api.WithTags("users"),    // Captured at runtime
    api.WithDeprecated("Use /v2/users"))  // Captured at runtime
```

The hybrid generator:

1. **Runtime provides**: 
   - POST /users → CreateUser
   - Request/Response types
   - Basic auth requirement
   - Tags: ["users"]
   - Deprecation info

2. **AST provides**:
   - Handler description: "Creates a new user account..."
   - Field descriptions: "Username must be unique..."
   - Deprecation notice from comments

3. **Combined output** includes everything

## Benefits of Simplified Approach

1. **Clear Separation**: Runtime handles behavior, AST handles documentation
2. **No Duplication**: Options are captured once at runtime
3. **Simpler AST Logic**: Only parse comments, not code patterns
4. **More Reliable**: No need to detect option usage patterns
5. **Easier to Maintain**: Less AST parsing code

## CLI Tool Design

```bash
# Generate OpenAPI spec (builds internally)
gork openapi generate \
  --build ./cmd/app \
  --source . \
  --output openapi.json \
  --title "My API" \
  --version "1.0.0"

# Or use configuration file
gork openapi generate --config .gork.yml

# Quick generation with defaults
gork openapi generate
```

Configuration file example (`.gork.yml`):

```yaml
openapi:
  build: ./cmd/app
  source: .
  output: ./docs/openapi.json
  title: My API
  version: 1.0.0
```

## Advanced Features (Documentation Only)

### 1. Example Detection from Comments

```go
// UserResponse represents a user in the system.
//
// Example:
//   {
//     "id": "123",
//     "username": "johndoe"
//   }
type UserResponse struct {
    ID       string `json:"id"`
    Username string `json:"username"`
}
```

### 2. Deprecation from Comments

```go
// Deprecated: This type will be removed in v2.0.
// Use UserResponseV2 instead.
type UserResponse struct {
    // ...
}
```

### 3. Field Documentation

```go
type User struct {
    // ID is the unique identifier.
    // Format: UUID v4
    // Example: "550e8400-e29b-41d4-a716-446655440000"
    ID string `json:"id"`
}
```

## Migration Path

1. **Update RouteInfo serialization** to ensure Options are exported
2. **Implement DocExtractor** for comment parsing
3. **Create hybrid generator** that combines both
4. **Test with examples** to ensure output quality
5. **Deprecate AST route detection**

## Implementation Plan

### Phase 1: Core Infrastructure (Week 1-2)
- [ ] Create `pkg/api/export.go` with `EnableOpenAPIExport()` and `ExportOpenAPIAndExit()`
- [ ] Implement `pkg/api/openapi/doc_extractor.go` for AST comment parsing
- [ ] Add JSON serialization support to RouteRegistry
- [ ] Create basic test suite for comment extraction

### Phase 2: CLI Tool Development (Week 3-4)
- [ ] Create `cmd/gork` with cobra command structure
- [ ] Implement `openapi generate` subcommand
- [ ] Add build automation (compile with openapi tag)
- [ ] Add configuration file support (`.gork.yml`)
- [ ] Implement output formatting (JSON/YAML)

### Phase 3: Generator Implementation (Week 5-6)
- [ ] Create `pkg/api/openapi/generator.go` combining runtime + AST
- [ ] Implement schema generation with validation tag mapping
- [ ] Add example extraction from comments
- [ ] Handle special types (time.Time, UUID, etc.)
- [ ] Add comprehensive test coverage

### Phase 4: Testing & Documentation (Week 7-8)
- [ ] Update examples to use new approach
- [ ] Create migration guide from AST-only tool
- [ ] Performance testing with large codebases
- [ ] Write user documentation

### Phase 5: Release & Deprecation (Week 9-10)
- [ ] Alpha release with select users
- [ ] Gather feedback and fix issues
- [ ] Official release
- [ ] Deprecate old AST-based tool

## Conclusion

This simplified hybrid approach leverages the existing runtime infrastructure for all behavioral metadata (routes, security, tags) while using AST only for what it does best: extracting documentation from comments. This results in a more maintainable, reliable, and efficient solution. 