# pkg/api - Convention Over Configuration HTTP Handler

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg?flag=pkg%2Fapi)](https://codecov.io/gh/gork-labs/gork/tree/main/pkg/api)

This package provides a Convention Over Configuration HTTP handler adapter that uses structured request types to eliminate the need for parameter location tags.

## Installation

```bash
go get github.com/gork-labs/gork/pkg/api
```

## Convention Over Configuration

Instead of using tags to specify parameter locations, use structured request types with standard sections: `Query`, `Body`, `Path`, `Headers`, and `Cookies`.

### Basic Example

```go
package main

import (
    "context"
    "net/http"
    "github.com/gork-labs/gork/pkg/api"
)

// User represents the user data structure
type User struct {
    // ID is the unique identifier for the user
    ID    string `gork:"id"`
    // Name is the user's full name
    Name  string `gork:"name"`
    // Email is the user's email address
    Email string `gork:"email"`
}

// Convention Over Configuration request structure
type CreateUserRequest struct {
    Body struct {
        // Name is the user's full name
        Name  string `gork:"name" validate:"required,min=3"`
        // Email is the user's email address
        Email string `gork:"email" validate:"required,email"`
    }
}

type CreateUserResponse struct {
    Body User
}

// Implement your business logic
func CreateUser(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
    return &CreateUserResponse{
        Body: User{
            ID:    "user-123",
            Name:  req.Body.Name,
            Email: req.Body.Email,
        },
    }, nil
}

func main() {
    // Create convention handler
    factory := api.NewConventionHandlerFactory()
    adapter := &api.HTTPParameterAdapter{}
    handler, _ := factory.CreateHandler(adapter, CreateUser)
    
    http.HandleFunc("/users", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Error Handling

The adapter automatically handles errors and returns appropriate HTTP responses:

```go
func GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    user, err := db.GetUser(req.Path.ID)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, &api.ErrorResponse{Error: "User not found"}
        }
        return nil, err // 500 Internal Server Error
    }
    return &GetUserResponse{Body: user}, nil
}
```

### Request Structure

Use structured sections to organize parameters by their HTTP location:

```go
type UpdateUserRequest struct {
    Path struct {
        // UserID is the unique identifier for the user to update
        UserID string `gork:"user_id" validate:"required,uuid"`
    }
    Query struct {
        // Notify determines if notifications should be sent
        Notify bool `gork:"notify"`
    }
    Headers struct {
        // Version specifies the API version for the request
        Version int `gork:"X-User-Version"`
    }
    Body struct {
        // Name is the updated user's full name
        Name  string `gork:"name" validate:"omitempty,min=3,max=100"`
        // Email is the updated user's email address
        Email string `gork:"email" validate:"omitempty,email"`
    }
}
```

### Mixed Parameter Example

All parameter types in one request:

```go
// POST /users/123?notify=true
// Headers: X-User-Version: 2
// Body: {"name": "John", "email": "john@example.com"}

type UpdateUserRequest struct {
    Path struct {
        // UserID comes from the URL path parameter
        UserID string `gork:"user_id"`
    }
    Query struct {
        // Notify comes from the query string
        Notify bool `gork:"notify"`
    }
    Headers struct {
        // Version comes from HTTP headers
        Version int `gork:"X-User-Version"`
    }
    Body struct {
        // Name comes from the JSON request body
        Name  string `gork:"name"`
        // Email comes from the JSON request body
        Email string `gork:"email"`
    }
}
```

### Context Usage

The adapter passes through the HTTP request context:

```go
func GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    // Access request-scoped values
    userID := ctx.Value("userID").(string)
    
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue processing
    }
    
    return fetchUser(ctx, req.Path.ID)
}
```

## Features

- **Convention Over Configuration**: No need for parameter location tags
- **Type Safety**: Compile-time type checking for requests and responses
- **Automatic Validation**: Built-in request validation using gork tags
- **Error Handling**: Consistent error responses with proper HTTP status codes
- **Structured Requests**: Clear separation of parameters by HTTP location
- **Context Propagation**: Full support for context cancellation and values
- **Framework Agnostic**: Works with any router that accepts `http.HandlerFunc`

## Handler Signature

Handlers must follow this signature:

```go
func HandlerName(ctx context.Context, req RequestType) (*ResponseType, error)
```

Where:
- `ctx` is the request context
- `req` is your request type with convention sections (validated automatically)
- `ResponseType` is your response type with convention sections (pointer)
- `error` is for error handling

## OpenAPI Integration

This adapter automatically generates OpenAPI specifications from convention-based request/response structures using the gork CLI tool.

## Examples

See the [examples](../../examples/) directory for complete working examples with different web frameworks.

## License

MIT License - see the root [LICENSE](../../LICENSE) file for details.