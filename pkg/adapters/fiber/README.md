# Fiber Adapter

This adapter provides integration between [Fiber](https://gofiber.io/) and the Gork API framework, enabling type-safe route registration and automatic OpenAPI spec generation.

## Installation

```bash
go get github.com/gork-labs/gork/pkg/adapters/fiber
```

## Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/gofiber/fiber/v2"
    "github.com/gork-labs/gork/pkg/adapters/fiber"
    "github.com/gork-labs/gork/pkg/api"
)

// User represents the user data structure
type User struct {
    // ID is the unique identifier for the user
    ID   string `gork:"id"`
    // Name is the user's full name
    Name string `gork:"name"`
}

// Convention Over Configuration request structure
type GetUserRequest struct {
    Path struct {
        // UserID is the unique identifier for the user
        UserID string `gork:"userId" validate:"required,uuid"`
    }
}

type GetUserResponse struct {
    Body User
}

func GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    return &GetUserResponse{
        Body: User{
            ID:   req.Path.UserID,
            Name: "John Doe",
        },
    }, nil
}

func main() {
    app := fiber.New()
    router := fiber.NewRouter(app)

    // Register typed routes
    router.Get("/users/{userId}", GetUser,
        api.WithTags("users"),
        api.WithBearerTokenAuth(),
    )

    // Serve API documentation
    router.DocsRoute("/docs/*", api.DocsConfig{
        Title: "My API Documentation",
    })

    log.Fatal(app.Listen(":8080"))
}
```

## Features

- **Type-safe handlers**: Automatic request/response marshaling with validation
- **Path parameters**: Extract path parameters using struct tags
- **Query parameters**: Extract query parameters from the request
- **Headers**: Access request headers through the adapter
- **Cookies**: Read HTTP cookies from requests
- **Documentation**: Serve interactive API documentation with multiple UI options
- **OpenAPI generation**: Automatic OpenAPI 3.1 spec generation from handler types

## Path Parameter Mapping

Gork uses `{param}` syntax which is automatically converted to Fiber's `:param` syntax:

```go
// Gork route definition
router.Get("/users/{userId}/posts/{postId}", handler)

// Becomes Fiber route
app.Get("/users/:userId/posts/:postId", fiberHandler)
```

## Documentation UIs

The adapter supports multiple documentation UIs:

```go
import "github.com/gork-labs/gork/pkg/api"

// Stoplight Elements (default)
router.DocsRoute("/docs/*", api.DocsConfig{
    UITemplate: api.StoplightUITemplate,
})

// Swagger UI
router.DocsRoute("/docs/*", api.DocsConfig{
    UITemplate: api.SwaggerUITemplate,
})

// Redoc
router.DocsRoute("/docs/*", api.DocsConfig{
    UITemplate: api.RedocUITemplate,
})
```

## Router Groups

Create sub-routers with shared prefixes:

```go
app := fiber.New()
router := fiber.NewRouter(app)

// API v1 group
v1 := router.Group("/api/v1")
v1.Get("/users", ListUsers)
v1.Post("/users", CreateUser)

// API v2 group
v2 := router.Group("/api/v2")
v2.Get("/users", ListUsersV2)
```

## Middleware Integration

Use Fiber's native middleware with the router:

```go
import "github.com/gofiber/fiber/v2/middleware/cors"

app := fiber.New()

// Apply CORS middleware
app.Use(cors.New(cors.Config{
    AllowOrigins: "https://example.com",
    AllowHeaders: "Origin, Content-Type, Accept, Authorization",
}))

router := fiber.NewRouter(app)
// Register routes...
```

## Error Handling

The adapter integrates with Gork's error handling system:

```go
func GetUser(ctx context.Context, req GetUserRequest) (User, error) {
    if req.UserID == "" {
        return User{}, api.NewValidationError("user_id is required")
    }

    // Your business logic here
    return user, nil
}
```

## Advanced Configuration

```go
router.DocsRoute("/docs/*", api.DocsConfig{
    Title:       "My API",
    OpenAPIPath: "/openapi.json", // Path to serve OpenAPI spec
    SpecFile:    "./openapi.json", // Use pre-generated spec file
    UITemplate:  api.StoplightUITemplate,
})
```

## Testing

The adapter is thoroughly tested and provides utilities for testing your handlers:

```go
func TestMyHandler(t *testing.T) {
    app := fiber.New()
    router := fiber.NewRouter(app)

    router.Get("/test", MyHandler)

    req := httptest.NewRequest("GET", "/test", nil)
    resp, err := app.Test(req)
    // Assert response...
}
```

## Complete Example

See the [examples directory](../../../examples/) for a complete working example with multiple handlers, authentication, and documentation.
