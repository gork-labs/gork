# Project Structure

## Monorepo Organization

This is a Go workspace monorepo with independent modules. Each module has its own `go.mod` and can be versioned separately.

## Directory Layout

```
gork/
├── cmd/                    # CLI applications
│   ├── gork/              # Main OpenAPI generation tool
│   └── lintgork/          # Convention compliance linter
├── pkg/                   # Public libraries (importable by users)
│   ├── api/               # Core HTTP handler and OpenAPI generation
│   ├── adapters/          # Framework-specific adapters
│   │   ├── stdlib/        # Standard library adapter
│   │   ├── gin/           # Gin framework adapter
│   │   ├── echo/          # Echo framework adapter
│   │   ├── chi/           # Chi router adapter
│   │   ├── fiber/         # Fiber framework adapter
│   │   └── gorilla/       # Gorilla Mux adapter
│   ├── unions/            # Type-safe union types
│   ├── gorkson/           # JSON marshaling utilities
│   └── webhooks/          # Webhook handling (e.g., Stripe)
├── internal/              # Private implementation details
│   ├── cli/               # CLI implementation
│   └── lintgork/          # Linter implementation
├── examples/              # Complete example API
│   ├── handlers/          # Example HTTP handlers
│   ├── cmd/               # Example commands
│   └── routes.go          # Route registration examples
├── scripts/               # Build and development automation
├── docs/                  # Documentation and specifications
└── .kiro/                 # Kiro IDE configuration
```

## Module Structure Conventions

### Request/Response Patterns
```go
// Convention Over Configuration structure
type HandlerRequest struct {
    Path struct {
        UserID string `gork:"userId" validate:"required,uuid"`
    }
    Query struct {
        Limit  int  `gork:"limit" validate:"min=1,max=100"`
        Offset int  `gork:"offset" validate:"min=0"`
    }
    Body struct {
        Name  string `gork:"name" validate:"required,min=1"`
        Email string `gork:"email" validate:"required,email"`
    }
    Headers struct {
        Authorization string `gork:"Authorization" validate:"required"`
    }
    Cookies struct {
        SessionID string `gork:"session_id"`
    }
}

type HandlerResponse struct {
    Body ResponseType
}
```

### Handler Signatures
```go
// Standard handler signature
func HandlerName(ctx context.Context, req RequestType) (*ResponseType, error)

// Error-only handler (no response body)
func HandlerName(ctx context.Context, req RequestType) error
```

## File Naming Conventions

- **Handlers**: `{resource}_{action}.go` (e.g., `users_create.go`)
- **Tests**: `{filename}_test.go` with comprehensive coverage
- **Adapters**: Framework name (e.g., `gin.go`, `echo.go`)
- **Examples**: Descriptive names in `examples/handlers/`

## Code Organization Principles

### Package Responsibilities
- **`pkg/api`**: Core framework logic, OpenAPI generation, validation
- **`pkg/adapters/*`**: Framework-specific HTTP routing integration
- **`pkg/unions`**: Type-safe union type implementations
- **`internal/*`**: Implementation details not exposed to users
- **`cmd/*`**: Executable CLI tools
- **`examples/`**: Working examples and integration tests

### Testing Structure
- Each module maintains 100% test coverage
- Tests are co-located with source files
- Integration tests in `examples/` demonstrate real usage
- Coverage reports generated per module

### Documentation Integration
- Go comments become OpenAPI field descriptions
- Struct tags define validation rules and JSON field names
- Examples serve as living documentation
- OpenAPI specs auto-generated from source code

## Import Patterns

```go
// Framework adapter usage
import "github.com/gork-labs/gork/pkg/adapters/stdlib"

// Core API functionality
import "github.com/gork-labs/gork/pkg/api"

// Union types for polymorphic responses
import "github.com/gork-labs/gork/pkg/unions"
```