# openapi-gen - OpenAPI 3.1.0 Code Generator for Go

A powerful tool that generates OpenAPI 3.1.0 specifications from Go source code by analyzing struct tags, handler signatures, and route definitions.

## Installation

```bash
go install github.com/gork-labs/gork/tools/openapi-gen/cmd/openapi-gen@latest
```

## Features

- **Automatic Schema Generation**: Extracts schemas from Go structs with JSON tags
- **Validator Tag Support**: Maps go-playground/validator tags to OpenAPI constraints
- **Union Type Support**: Handles discriminated unions with proper oneOf schemas
- **Multiple Framework Support**: Works with Gin, Echo, Chi, Gorilla Mux, Fiber, and standard library
- **Type-Safe Handlers**: Designed for use with the api adapter package
- **Path Parameter Detection**: Automatically extracts path parameters from routes

## Usage

### Basic Usage

```bash
# Generate OpenAPI spec for current directory
openapi-gen -o openapi.json

# Specify input directories and route files
openapi-gen -i ./handlers -i ./models -r ./routes/routes.go -o openapi.yaml -f yaml

# With custom API metadata
openapi-gen -t "My API" -v "2.0.0" -d "My awesome API" -o spec.json
```

### Command Line Options

```
Flags:
  -i, --input strings        Input directories to scan (default [.])
  -r, --routes strings       Route registration files
  -o, --output string        Output file path (default "openapi.json")
  -f, --format string        Output format: json or yaml (default "json")
  -t, --title string         API title (default "API")
  -v, --version string       API version (default "1.0.0")
  -d, --description string   API description
  --generate-union-accessors Generate accessor methods for union types
  --union-output string      Output file for union accessors
  --colocated               Generate accessors alongside source files
```

### Handler Pattern

The generator recognizes handlers with this signature:

```go
func HandlerName(ctx context.Context, req RequestType) (*ResponseType, error)
```

Example:

```go
package handlers

import (
    "context"
    "github.com/gork-labs/gork/pkg/api"
)

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=3"`
    Email string `json:"email" validate:"required,email"`
}

type UserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
    // Implementation
    return &UserResponse{
        ID:    "user-123",
        Name:  req.Name,
        Email: req.Email,
    }, nil
}
```

### Route Registration

Register routes using your framework of choice:

```go
// Gin
router.POST("/users", api.HandlerFunc(handlers.CreateUser))

// Echo
e.POST("/users", api.HandlerFunc(handlers.CreateUser))

// Chi
r.Post("/users", api.HandlerFunc(handlers.CreateUser))

// Standard library
http.HandleFunc("/users", api.HandlerFunc(handlers.CreateUser))
```

### Struct Tags

#### JSON Tags
```go
type User struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Email    string   `json:"email"`
    Password string   `json:"-"`              // Excluded from API
    Tags     []string `json:"tags,omitempty"` // Optional field
}
```

#### Validator Tags
```go
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required,min=3,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=18,max=120"`
    Website  string `json:"website" validate:"omitempty,url"`
    Password string `json:"password" validate:"required,min=8"`
}
```

#### OpenAPI Tags
```go
type PaginatedRequest struct {
    Page     int    `json:"page" openapi:"in=query" validate:"min=1"`
    PageSize int    `json:"page_size" openapi:"in=query" validate:"min=1,max=100"`
    APIKey   string `json:"-" openapi:"name=X-API-Key,in=header" validate:"required"`
}
```

### Union Types

The generator supports discriminated unions:

```go
import "github.com/gork-labs/gork/pkg/unions"

type PaymentMethod unions.Union3[CreditCard, BankAccount, PayPal]

type PaymentRequest struct {
    Amount int            `json:"amount" validate:"required,min=1"`
    Method PaymentMethod  `json:"method" validate:"required"`
}
```

This generates an OpenAPI oneOf schema with proper discriminators.

### Generating Union Accessors

Generate type-safe accessor methods for your union types:

```bash
# Generate in a single file
openapi-gen -i ./models --generate-union-accessors --union-output ./models/unions_gen.go

# Generate alongside source files
openapi-gen -i ./models --generate-union-accessors --colocated
```

This creates methods like:

```go
func (u *PaymentMethod) IsCreditCard() bool
func (u *PaymentMethod) CreditCard() CreditCard
func (u *PaymentMethod) IsBankAccount() bool
func (u *PaymentMethod) BankAccount() BankAccount
```

## Supported Validators

The tool maps go-playground/validator tags to OpenAPI:

- `required` → required field
- `email` → format: email
- `uuid` → format: uuid
- `uri/url` → format: uri
- `min/max` → minimum/maximum (numbers) or minLength/maxLength (strings)
- `len` → exact length constraint
- `oneof` → enum values
- `unique` → uniqueItems for arrays
- Pattern validators → pattern property

## Examples

See the [examples](../examples/) directory for complete examples including:
- Basic CRUD API
- Union types for payment methods
- Custom validators
- Multi-framework examples

## Limitations

- Only supports the specific handler signature pattern
- Requires explicit route registration (no magic comments)
- Union types limited to Union2, Union3, and Union4

## License

MIT License - see the root [LICENSE](../LICENSE) file for details.