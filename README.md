# OpenAPI Generator for Go

A code-first OpenAPI 3.1.0 schema generator that extracts API documentation from Go source code using go-playground/validator tags for validation constraints.

## Features

- **Code-First Approach**: Generate OpenAPI specs directly from your Go code
- **Validator Tag Support**: Automatically maps go-playground/validator tags to OpenAPI constraints
- **Type-Safe Handlers**: Recognizes handler pattern `func(context.Context, RequestType) (ResponseType, error)`
- **Multi-Router Support**: Works with standard library, Gin, Echo, Gorilla Mux, Chi, and Fiber
- **Union Types**: Support for union types with discriminators (Union2, Union3, Union4)
- **Custom Validators**: Support for custom validation rules with documentation
- **Zero Annotations**: No special comments or tags required beyond standard struct tags

## Installation

```bash
go install github.com/example/openapi-gen/cmd/openapi-gen@latest
```

## Quick Start

1. Define your models with validator tags:

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Username string `json:"username" validate:"required,alphanum,min=3,max=50"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age,omitempty" validate:"omitempty,gte=0,lte=150"`
    Role     string `json:"role" validate:"required,oneof=admin user moderator"`
}

type User struct {
    ID       string `json:"id" validate:"required,uuid"`
    Email    string `json:"email" validate:"required,email,max=255"`
    Username string `json:"username" validate:"required,alphanum,min=3,max=50"`
    Age      int    `json:"age,omitempty" validate:"omitempty,gte=0,lte=150"`
    Role     string `json:"role" validate:"required,oneof=admin user moderator"`
}
```

2. Create handlers with typed signatures:

```go
package handlers

import (
    "context"
    "github.com/google/uuid"
)

func CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    // Validate and create user
    user := &User{
        ID:       uuid.New().String(),
        Email:    req.Email,
        Username: req.Username,
        Age:      req.Age,
        Role:     req.Role,
    }
    // Save to database...
    return user, nil
}

func GetUser(ctx context.Context, req GetUserRequest) (*User, error) {
    // req.ID is automatically extracted from path parameter
    // Fetch user from database...
    return &User{
        ID:       req.ID,
        Email:    "user@example.com",
        Username: "johndoe",
        Role:     "user",
    }, nil
}

func ListUsers(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error) {
    // req contains query parameters like page, limit, etc.
    // Fetch users from database...
    return &ListUsersResponse{
        Users: []User{},
        Total: 0,
    }, nil
}
```

3. Register routes with API wrapper, tags, and authentication:

```go
package main

import (
    "net/http"
    "github.com/example/openapi-gen/pkg/api"
    "github.com/example/myapp/handlers"
)

func setupRoutes() {
    mux := http.NewServeMux()
    
    // Public endpoints
    mux.HandleFunc("POST /api/v1/auth/login", 
        api.HandlerFunc(handlers.Login, api.WithTags("auth")))
    
    // Protected endpoints with authentication
    mux.HandleFunc("POST /api/v1/users", 
        api.HandlerFunc(handlers.CreateUser, 
            api.WithTags("users"), 
            api.WithBearerTokenAuth("admin")))
    
    mux.HandleFunc("GET /api/v1/users/{id}", 
        api.HandlerFunc(handlers.GetUser, 
            api.WithTags("users"), 
            api.WithBearerTokenAuth("read:users")))
    
    mux.HandleFunc("GET /api/v1/users", 
        api.HandlerFunc(handlers.ListUsers, 
            api.WithTags("users"), 
            api.WithAPIKeyAuth()))
    
    // Basic auth example
    mux.HandleFunc("DELETE /api/v1/users/{id}", 
        api.HandlerFunc(handlers.DeleteUser, 
            api.WithTags("users"), 
            api.WithBasicAuth()))
    
    http.ListenAndServe(":8080", mux)
}
```

4. Generate OpenAPI spec:

```bash
openapi-gen -i ./handlers -r ./main.go -o openapi.json -t "User Management API" -v "1.0.0"
```

## CLI Options

```bash
openapi-gen [flags]

Flags:
  -i, --input strings              Input directories to scan (default [.])
  -r, --routes strings             Route registration files
  -o, --output string              Output file path (default "openapi.json")
  -t, --title string               API title (default "API")
  -v, --version string             API version (default "1.0.0")
  -f, --format string              Output format: json or yaml (default "json")
  -d, --description string         API description
  --generate-union-accessors       Generate accessor methods for user-defined union types
  --union-output string            Output file for generated union accessors (defaults to union_accessors.go)
  --colocated                      Generate accessor files alongside source files with _goapi_gen.go suffix
```

## Validator Tag Mapping

The generator automatically maps go-playground/validator tags to OpenAPI constraints:

| Validator Tag | OpenAPI Property | Example |
|--------------|------------------|---------|
| `required` | required field | `validate:"required"` |
| `email` | format: email | `validate:"email"` |
| `url` | format: uri | `validate:"url"` |
| `uuid` | format: uuid | `validate:"uuid"` |
| `min=n` | minLength (string) or minimum (number) | `validate:"min=3"` |
| `max=n` | maxLength (string) or maximum (number) | `validate:"max=100"` |
| `gte=n` | minimum | `validate:"gte=0"` |
| `lte=n` | maximum | `validate:"lte=150"` |
| `oneof=...` | enum | `validate:"oneof=admin user moderator"` |
| `alpha` | pattern: ^[a-zA-Z]+$ | `validate:"alpha"` |
| `alphanum` | pattern: ^[a-zA-Z0-9]+$ | `validate:"alphanum"` |
| `numeric` | pattern: ^[0-9]+$ | `validate:"numeric"` |
| `e164` | format: e164 (phone) | `validate:"e164"` |
| `len=n` | minLength & maxLength | `validate:"len=3"` |
| `dive` | validates array items | `validate:"dive,min=1"` |

## Custom Validators

Register custom validators and they'll be documented in the OpenAPI spec:

```go
// In your code
gen.RegisterCustomValidator("username", "Username must be alphanumeric with optional underscores")
gen.RegisterCustomValidator("strongpassword", "Password must contain uppercase, lowercase, number, and special character")
```


## Union Types

The generator supports union types for polymorphic APIs:

```go
// Traditional union types (Union2, Union3, Union4)
type PaymentData unions.Union3[CreditCard, BankTransfer, PayPal]
```

### Discriminator mapping

If your union members implement a common field (e.g. `Type string \`json:"type"\``) or you mark a field with the tag `openapi:"discriminator:<fieldName>"`, the generator will automatically add an OpenAPI discriminator section:

```yaml
discriminator:
  propertyName: type
  mapping:
    creditCard:   #/components/schemas/CreditCard
    bankTransfer: #/components/schemas/BankTransfer
    payPal:       #/components/schemas/PayPal
```

This lets Swagger-UI and other tools unmarshal to the correct concrete schema. If no explicit tag is found but all union members share a common field name, the generator will pick that field automatically.

### Accessor helpers

Generated methods include:
- `IsOptionName()` - Check if a specific option is set
- `OptionName()` - Get the value of a specific option  
- `SetOptionName(value)` - Set the union to contain a specific option
- `Value()` - Get whichever value is currently set
- `NewTypeFromOption(value)` - Constructor function to create a union with a specific option

## Handler Patterns

The generator recognizes these handler signatures:

```go
// Standard pattern
func Handler(ctx context.Context, req RequestType) (*ResponseType, error)

// No response body
func Handler(ctx context.Context, req RequestType) error

// Query parameters (GET/DELETE)
func ListUsers(ctx context.Context, req ListRequest) (*ListResponse, error)

// Path parameters
type GetUserRequest struct {
    ID string `json:"id" validate:"required,uuid"` // Will be path param
}
```

## Example Generated Spec

Run `make example-spec` (or `openapi-gen -i examples/handlers -r examples/routes.go -o examples/openapi.json`) to produce an updated **examples/openapi.json**. A fresh version is committed in the repo for quick inspection.

## Examples

See the `examples/` directory for a complete example including:

- User management API
- Product catalog API  
- Order processing API
- Payment processing with union types
- Custom validators
- Route registration

To generate the OpenAPI spec for the examples:

```bash
openapi-gen -i examples/handlers -i examples/models -i examples/validators -r examples/routes/routes.go -o examples/openapi.json
```

## Union Types and Accessor Generation

The generator supports union types (Union2, Union3, Union4) and can optionally generate type-safe accessor methods for user-defined union type aliases.

### Defining Union Types

```go
// Define a union type alias
type PaymentMethod unions.Union2[CreditCard, BankAccount]

// Use in your request structs
type PaymentRequest struct {
    UserID string        `json:"userId" validate:"required"`
    Method PaymentMethod `json:"method" validate:"required"`
}
```

### Generating Union Accessors

Use the `--generate-union-accessors` flag to generate helper methods:

```bash
# Generate a single file with all accessors
openapi-gen -i ./handlers --generate-union-accessors --union-output ./handlers/union_accessors.go

# Generate co-located files (recommended) - creates *_goapi_gen.go files alongside source files
openapi-gen -i ./handlers --generate-union-accessors --colocated
```

This generates methods like:

```go
// Create using constructor function
payment := NewPaymentMethodFromCreditCard(&CreditCard{
    Number: "4111111111111111",
})

// Check which type is set
if payment.IsCreditCard() {
    card := payment.CreditCard()
    // Use credit card...
}

// Change to a different type using setter
payment.SetBankAccount(&BankAccount{
    AccountNumber: "123456789",
})

// Get the active value
switch v := payment.Value().(type) {
case *CreditCard:
    // Process credit card
case *BankAccount:
    // Process bank account
}
```

Generated methods include:
- `IsOptionName()` - Check if a specific option is set
- `OptionName()` - Get the value of a specific option  
- `SetOptionName(value)` - Set the union to contain a specific option
- `Value()` - Get whichever value is currently set
- `NewTypeFromOption(value)` - Constructor function to create a union with a specific option

## How It Works

1. **AST Parsing**: Analyzes Go source files to extract type definitions and handler functions
2. **Validator Mapping**: Converts go-playground/validator tags to OpenAPI constraints
3. **Route Detection**: Identifies route registrations from various web frameworks
4. **Schema Generation**: Creates OpenAPI schemas with proper validation rules
5. **Operation Building**: Maps handlers to operations with request/response schemas

## Limitations

- Handlers must follow the pattern `func(context.Context, RequestType) (ResponseType, error)`
- Path parameters must be included in request structs
- Anonymous structs in handlers are supported but limited
- Complex validation rules (cross-field, conditional) are added to descriptions

## License

MIT