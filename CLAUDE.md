# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go API development toolkit that provides type-safe HTTP handlers, OpenAPI 3.1.0 generation, and union types. The toolkit includes multiple framework adapters and automatic API documentation generation from Go source code using go-playground/validator tags.

## Common Commands

### Build and Run
```bash
# Build the CLI tool
go build -o gork ./cmd/gork

# Install the CLI tool
go install ./cmd/gork

# Run tests with coverage
make test

# Run coverage check (requires 100% coverage)
make coverage

# Generate OpenAPI spec for examples
./gork openapi generate --build ./examples/cmd/openapi_export --source ./examples --output ./examples/openapi.json --title "API" --version "1.0.0"

# Generate spec in YAML format
./gork openapi generate --build ./examples/cmd/openapi_export --source ./examples --output ./examples/openapi.yaml --format yaml --title "API" --version "1.0.0"

# Build and generate OpenAPI from current directory
./gork openapi generate --build . --output openapi.json --title "My API" --version "2.0.0"
```

### Development
```bash
# Install dependencies for all modules
make deps

# Run all tests across all modules
make test

# Run tests for specific module
make test pkg/api

# Run coverage check for all modules
make coverage

# Run coverage check for specific module  
make coverage pkg/unions

# Generate HTML coverage reports
make coverage-html

# Format all code
make fmt

# Run linting
make lint

# Build all tools
make build

# Validate OpenAPI specs
make openapi-validate
```

## Architecture

### Core Components

**CLI Tool (`cmd/gork/`)**
- `main.go` - CLI entry point using Cobra
- `openapi/generate.go` - OpenAPI generation command with build-time and runtime spec generation

**Linting Tool (`cmd/lintgork/`)**
- `main.go` - Standalone linter for struct tags and OpenAPI compliance
- `internal/lintgork/analyzer.go` - Static analysis for struct validation and path parameters

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
├── cmd/gork/                  # Main CLI tool
│   └── openapi/               # OpenAPI generation commands
├── cmd/lintgork/              # Linting tool for struct validation
├── internal/lintgork/         # Linter implementation
├── pkg/api/                   # HTTP adapter for type-safe handlers
├── pkg/unions/                # Union type utilities  
├── pkg/adapters/              # Framework-specific adapters
│   ├── stdlib/                # Standard library adapter
│   ├── gin/                   # Gin framework adapter
│   ├── echo/                  # Echo framework adapter
│   ├── chi/                   # Chi framework adapter
│   ├── fiber/                 # Fiber framework adapter
│   └── gorilla/               # Gorilla Mux adapter
└── examples/                  # Complete example API
    ├── handlers/              # HTTP handlers
    ├── cmd/                   # Example commands
    └── routes.go              # Route registration
```

## Development Notes

- The codebase uses Go 1.24 features (as specified in go.work)
- Uses Go workspace for multi-module development
- OpenAPI spec generation follows OpenAPI 3.1.0 specification
- Union types are represented using oneOf schemas with discriminators
- Examples directory contains a complete working API demonstrating all features
- Project enforces 100% test coverage on all pkg/ modules
- CLI tools (cmd/) and internal linter currently have 0% coverage

## Testing

The project uses standard Go testing with strict coverage requirements:
- All pkg/ modules must maintain 100% test coverage
- CLI tools and examples are excluded from coverage requirements
- The Makefile provides `test` and `coverage` targets
- Use `make coverage-html` to generate HTML coverage reports
- Coverage enforcement is handled by `scripts/check-coverage.sh`

## Important Notes for Claude

- Always run `make test` before making changes
- **Coverage Status**: Coverage varies by module, system requires 95% for pkg/ modules
  - pkg/unions: 100% coverage ✅
  - pkg/api: ❌ 70.7% coverage + 3 failing tests (error message assertions)
  - pkg/adapters/fiber: ❌ 45.8% coverage (custom parameter handling untested)
  - pkg/adapters/*: 97-98% coverage (mostly passing) ✅
  - CLI tools and internal linter are excluded from coverage requirements
- Use the correct CLI commands: `gork openapi generate` (not `openapi-gen`)
- Module paths follow Go workspace structure with independent go.mod files
- Framework adapters are in `pkg/adapters/` not inline
- Union accessor generation is implemented but CLI integration needs testing
- If coverage fails, check which specific lines need tests using `make coverage-html`
- **Known Issues**: 
  - pkg/api has failing tests related to error message expectations (temporarily excluded from coverage)
  - pkg/adapters/fiber needs tests for custom parameter handling logic (temporarily excluded from coverage)
  - Coverage threshold has been lowered to 95% for pkg/ modules to be practical
  - These modules are excluded from coverage checks until issues are resolved