# pkg/api - Type-Safe HTTP Handler Adapter

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg?flag=pkg%2Fapi)](https://codecov.io/gh/gork-labs/gork/tree/main/pkg/api)

This package provides a type-safe HTTP handler adapter that bridges between your business logic and HTTP transport layer.

## Installation

```bash
go get github.com/gork-labs/gork/pkg/api
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "net/http"
    "github.com/gork-labs/gork/pkg/api"
)

// Define request and response types
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=3"`
    Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Implement your business logic
func CreateUser(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
    // Your business logic here
    return &CreateUserResponse{
        ID:    "user-123",
        Name:  req.Name,
        Email: req.Email,
    }, nil
}

func main() {
    // Create HTTP handler
    handler := api.HandlerFunc(CreateUser)
    
    // Use with standard library
    http.HandleFunc("/users", handler)
    
    // Or with any router that accepts http.HandlerFunc
    http.ListenAndServe(":8080", nil)
}
```

### Error Handling

The adapter automatically handles errors and returns appropriate HTTP responses:

```go
func GetUser(ctx context.Context, req GetUserRequest) (*User, error) {
    user, err := db.GetUser(req.ID)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, api.NewError(http.StatusNotFound, "User not found")
        }
        return nil, err // 500 Internal Server Error
    }
    return user, nil
}
```

### Request Validation

The adapter automatically validates requests using struct tags:

```go
type UpdateUserRequest struct {
    ID    string `json:"id" validate:"required,uuid"`
    Name  string `json:"name" validate:"omitempty,min=3,max=100"`
    Email string `json:"email" validate:"omitempty,email"`
}
```

### Path Parameters

Path parameters are automatically extracted and merged into the request:

```go
type GetUserRequest struct {
    ID string `json:"id" validate:"required,uuid"` // From path: /users/{id}
}

// Router setup (example with chi)
r.Get("/users/{id}", api.HandlerFunc(GetUser))
```

### Context Usage

The adapter passes through the HTTP request context:

```go
func GetUser(ctx context.Context, req GetUserRequest) (*User, error) {
    // Access request-scoped values
    userID := ctx.Value("userID").(string)
    
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue processing
    }
    
    return fetchUser(ctx, req.ID)
}
```

## Features

- **Type Safety**: Compile-time type checking for requests and responses
- **Automatic Validation**: Built-in request validation using struct tags
- **Error Handling**: Consistent error responses with proper HTTP status codes
- **Path Parameters**: Automatic extraction and merging of path parameters
- **Context Propagation**: Full support for context cancellation and values
- **Framework Agnostic**: Works with any router that accepts `http.HandlerFunc`

## Handler Signature

Handlers must follow this signature:

```go
func HandlerName(ctx context.Context, req RequestType) (*ResponseType, error)
```

Where:
- `ctx` is the request context
- `req` is your request type (validated automatically)
- `ResponseType` is your response type (pointer)
- `error` is for error handling

## OpenAPI Integration

This adapter is designed to work with the openapi-gen tool for automatic API documentation generation. The handler signature and struct tags are automatically parsed to generate OpenAPI specifications.

## Examples

See the [examples](../../examples/) directory for complete working examples with different web frameworks.

## License

MIT License - see the root [LICENSE](../../LICENSE) file for details.