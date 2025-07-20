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

### Parameter Handling

The adapter supports multiple parameter locations through the `openapi` tag:

#### Query Parameters

Query parameters are automatically parsed from the URL for GET and DELETE requests:

```go
type ListUsersRequest struct {
    // Using json tag (default behavior)
    Page     int    `json:"page" validate:"min=1"`
    PageSize int    `json:"page_size" validate:"min=1,max=100"`
    
    // Using openapi tag for explicit query parameters
    Filter   string `openapi:"filter,in=query"`
    Sort     string `openapi:"sort_by,in=query"` // Custom parameter name
}

// Usage: GET /users?page=1&page_size=20&filter=active&sort_by=name
```

#### Mixed Parameters

The same struct can have parameters from different sources:

```go
type UpdateUserRequest struct {
    // From path (handled by router)
    UserID   string `openapi:"userID,in=path"`
    
    // From request body (JSON)
    Name     string `json:"name"`
    Email    string `json:"email"`
    
    // From query string
    Notify   bool   `openapi:"notify,in=query"`
    
    // From headers (not automatically parsed by adapter)
    Version  int    `openapi:"X-User-Version,in=header"`
}
```

**Note**: The adapter automatically parses in this order:
1. **Headers** (for all HTTP methods) with `openapi:"name,in=header"`
2. **JSON body** (for POST/PUT/PATCH requests)
3. **Query parameters** (for all HTTP methods) with `in=query` or from `json` tags

This parsing order means:
- Headers are parsed first and can be overridden by body/query values
- For POST/PUT/PATCH: body values can be overridden by query parameters
- Path parameters must still be handled by your router

Example with all parameter types:
```go
// POST /users/123?notify=true
// Headers: X-User-Version: 2
// Body: {"name": "John", "email": "john@example.com"}

type UpdateUserRequest struct {
    UserID  string `openapi:"userID,in=path"`      // From router
    Version int    `openapi:"X-User-Version,in=header"` // Parsed from headers
    Name    string `json:"name"`                   // From JSON body
    Email   string `json:"email"`                  // From JSON body  
    Notify  bool   `openapi:"notify,in=query"`     // From query string
}
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
- **Multi-Source Parameters**: Automatic parsing from headers, body, and query string
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