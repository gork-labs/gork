# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an OpenAPI 3.1.0 code generator for Go that extracts API documentation from Go source code using go-playground/validator tags. The tool generates OpenAPI specs from type-safe handler functions without requiring annotations.

## Common Commands

### Build and Run
```bash
# Build the CLI tool
go build -o openapi-gen ./cmd/openapi-gen

# Run tests with coverage
make test

# Generate OpenAPI spec for examples
go run ./cmd/openapi-gen -i examples -r examples/routes/routes.go -o examples/openapi.json

# Generate spec with custom options
openapi-gen -i ./models -r ./routes.go -o openapi.json -t "My API" -v "2.0.0" -f yaml

# Generate union accessor methods (single file)
openapi-gen -i ./handlers --generate-union-accessors --union-output ./handlers/union_accessors.go

# Generate union accessor methods (co-located with source files)
openapi-gen -i ./handlers --generate-union-accessors --colocated
```

### Development
```bash
# Install dependencies 
go mod tidy

# Run all tests
go test ./... -v

# Run tests with coverage report
go test ./... -v -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html
```

## Architecture

### Core Components

**Generator (`internal/generator/`)**
- `generator.go` - Main orchestrator that coordinates parsing and generation
- `extractor.go` - AST parser for Go source files, extracts types and handlers  
- `route_detector.go` - Detects route registrations from various web frameworks
- `validator_mapper.go` - Maps go-playground/validator tags to OpenAPI constraints
- `types.go` - OpenAPI spec data structures
- `metadata.go` - Metadata extraction from struct tags and comments

**Union Type Support**
- `union_detector.go` - Detects union types (Union2/Union3/Union4)
- `union_schema_registry.go` - Manages union type schemas and discriminators
- `openapi_union.go` - Generates OpenAPI oneOf schemas for unions
- `union_accessors.go` - Generates accessor methods for user-defined union types

**API Adapter (`pkg/api/`)**
- `adapter.go` - HTTP handler wrapper that provides type safety and OpenAPI metadata

**Union Utilities (`pkg/unions/`)**
- `unions.go` - Runtime utilities for working with union types (Union2/Union3/Union4)

### Handler Pattern

The generator recognizes this specific handler signature:
```go
func HandlerName(ctx context.Context, req RequestType) (*ResponseType, error)
```

Key requirements:
- Must accept `context.Context` as first parameter
- Second parameter is the request type (struct with validator tags)
- Returns pointer to response type and error
- Request structs define both JSON body and path parameters

### Validator Tag Mapping

The system automatically converts go-playground/validator tags to OpenAPI constraints:
- `required` → required field
- `email` → format: email  
- `min=n`/`max=n` → minLength/maxLength (strings) or minimum/maximum (numbers)
- `oneof=...` → enum values
- `uuid` → format: uuid
- Custom validators → added to field descriptions

### Route Detection

Supports multiple web frameworks:
- Standard library (`http.ServeMux`)
- Gin, Echo, Gorilla Mux, Chi, Fiber
- Custom `api.HandlerFunc` wrapper for metadata

The route detector parses route registration files to map HTTP methods and paths to handler functions.

### Union Types

The system supports union types through the UnionN pattern:

```go
type PaymentData unions.Union3[CreditCard, BankTransfer, PayPal]
type LoginRequest unions.Union2[EmailLogin, PhoneLogin]
type AuthMethod unions.Union4[Password, OAuth, APIKey, Certificate]
```

The generator creates OpenAPI oneOf schemas with proper discriminators for these union types.

### Union Accessor Generation

The tool can optionally generate type-safe accessor methods for user-defined union types:

```go
// Define a union type alias
type PaymentMethod unions.Union2[CreditCard, BankAccount]

// Generated methods:
func (u *PaymentMethod) IsCreditCard() bool
func (u *PaymentMethod) CreditCard() CreditCard
func (u *PaymentMethod) IsBankAccount() bool
func (u *PaymentMethod) BankAccount() BankAccount
func (u *PaymentMethod) Value() interface{}
```

This feature is opt-in via the `--generate-union-accessors` flag.

## Module Structure

```
github.com/gork-labs/gork/
├── tools/openapi-gen/         # OpenAPI generator tool
│   ├── cmd/openapi-gen/       # CLI entry point
│   └── internal/              # Core generation logic
├── pkg/api/                   # HTTP adapter for type-safe handlers
├── pkg/unions/                # Union type utilities
└── examples/                  # Complete example API
    ├── handlers/              # HTTP handlers
    ├── cmd/                   # Example commands
    └── routes.go              # Route registration
```

## Development Notes

- The codebase uses Go 1.21+ features
- AST parsing is done using Go's built-in `go/ast` and `go/parser` packages
- OpenAPI spec generation follows OpenAPI 3.1.0 specification
- Union types are represented using oneOf schemas with discriminators
- Examples directory contains a complete working API demonstrating all features

## Testing

The project uses standard Go testing. The Makefile provides a `test` target that runs tests with coverage reporting. Coverage reports are generated as HTML files for easy viewing.