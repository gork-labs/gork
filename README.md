# Gork - Opinionated Convention Over Configuration OpenAPI Framework

[![CI](https://github.com/gork-labs/gork/workflows/CI/badge.svg)](https://github.com/gork-labs/gork/actions)
[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg)](https://codecov.io/gh/gork-labs/gork)
[![Go Report Card](https://goreportcard.com/badge/github.com/gork-labs/gork)](https://goreportcard.com/report/github.com/gork-labs/gork)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Gork** is an opinionated convention over configuration OpenAPI framework for Go that provides type-safe HTTP handlers, automatic OpenAPI 3.1.0 generation, and union types. Built for developer productivity and business development efficiency.

## 🚀 Quick Start

Here's a simple example showing how to create a type-safe API with automatic OpenAPI generation:

```go
package main

import (
    "context"
    "net/http"
    
    "github.com/gork-labs/gork/pkg/adapters/stdlib"
    "github.com/gork-labs/gork/pkg/api"
)

// Request follows convention: Query, Body, Path, Headers, Cookies sections
type GetUserRequest struct {
    Path struct {
        // UserID is the unique identifier for the user
        UserID string `gork:"userId" validate:"required,uuid"`
    }
    Query struct {
        // IncludeProfile determines if user profile data should be included
        IncludeProfile bool `gork:"include_profile"`
    }
}

// User represents the user data structure
type User struct {
    // ID is the unique identifier for the user
    ID       string `gork:"id"`
    // Username is the user's chosen display name
    Username string `gork:"username"`
    // Email is the user's email address
    Email    string `gork:"email"`
}

// Response with typed body
type GetUserResponse struct {
    Body User
}

// Type-safe handler with strict signature
func GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    return &GetUserResponse{
        Body: User{
            ID:       req.Path.UserID,
            Username: "john_doe",
            Email:    "john@example.com",
        },
    }, nil
}

func main() {
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux)
    
    // Register route with automatic OpenAPI metadata extraction
    router.Get("/users/{userId}", GetUser, api.WithTags("users"))
    
    // Serve interactive API documentation
    router.DocsRoute("/docs/*")
    
    http.ListenAndServe(":8080", mux)
    // Now available:
    // - Interactive docs: http://localhost:8080/docs/
    // - OpenAPI spec: http://localhost:8080/openapi.json
}
```

**That's it!** Your API now has:
- ✅ Type-safe request/response handling
- ✅ Automatic validation using `validate` tags  
- ✅ OpenAPI 3.1.0 spec generation at `/openapi.json`
- ✅ Interactive docs at `/docs/`
- ✅ No boilerplate, just business logic

> **💡 Documentation Magic**: Notice how the Go comments above struct fields automatically become field descriptions in your OpenAPI documentation! No need to maintain separate documentation - your code comments become live API docs.

## Repository Structure

```
gork/
├── cmd/
│   ├── gork/          # Main CLI tool for OpenAPI generation
│   └── lintgork/      # Custom linter for struct validation and OpenAPI compliance
├── pkg/
│   ├── api/           # HTTP handler adapter and OpenAPI generation
│   ├── adapters/      # Framework-specific adapters
│   │   ├── chi/       # Chi router adapter
│   │   ├── echo/      # Echo framework adapter  
│   │   ├── fiber/     # Fiber framework adapter
│   │   ├── gin/       # Gin framework adapter
│   │   ├── gorilla/   # Gorilla Mux adapter
│   │   └── stdlib/    # Standard library adapter
│   └── unions/        # Type-safe union types for Go
├── internal/
│   ├── cli/           # CLI implementation
│   └── lintgork/      # Linter implementation
├── examples/          # Complete example API
│   ├── handlers/      # Example HTTP handlers
│   ├── cmd/           # Example commands
│   └── routes.go      # Route registration
├── scripts/           # Build and development scripts
└── Makefile           # Build and test automation
```

## 🛠️ Core Features

### Convention Over Configuration
- **Structured Requests**: Use standard sections `Query`, `Body`, `Path`, `Headers`, `Cookies`
- **Type Safety**: Strict handler signatures with compile-time validation
- **Self-Documenting**: Request structs serve as live API documentation
- **Consistent Naming**: `gork` tags replace framework-specific tags

### Automatic OpenAPI Generation
- **OpenAPI 3.1.0**: Full specification generation from Go source code
- **Validator Integration**: `go-playground/validator` tags become OpenAPI constraints
- **Union Types**: Type-safe variants with `oneOf` schemas and discriminators
- **Multi-Framework**: Works with Gin, Echo, Chi, Gorilla Mux, Fiber, stdlib

### Developer Experience
- **Zero Boilerplate**: Focus on business logic, not API plumbing
- **Interactive Docs**: Built-in documentation server
- **Static Analysis**: Custom linter ensures convention compliance
- **100% Coverage**: Quality-first development with strict testing

## 📦 CLI Tools

### gork - OpenAPI Generator

```bash
go install github.com/gork-labs/gork/cmd/gork@latest

# Generate OpenAPI spec from your handlers
gork openapi generate --build ./cmd/server --source ./handlers --output openapi.json

# With custom metadata and YAML output  
gork openapi generate --source ./api --output spec.yaml --format yaml \
  --title "My API" --version "2.0.0"
```

### lintgork - Convention Linter

```bash
go install github.com/gork-labs/gork/cmd/lintgork@latest

# Validate convention compliance
lintgork ./...

# Integrates with golangci-lint
```

## 🎯 Rules Engine

Gork includes a powerful, lightweight rules engine that enables business logic validation through simple struct tags. Rules are perfect for authorization, ownership checks, and complex business validations that go beyond standard validation tags.

### Basic Rule Usage

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    "github.com/gork-labs/gork/pkg/adapters/stdlib"
    "github.com/gork-labs/gork/pkg/api"
    "github.com/gork-labs/gork/pkg/rules"
)

// Example ownership database (in real apps, this would be your database)
var itemOwners = map[string]string{
    "item-123": "alice",
    "item-456": "bob",
}

var userRoles = map[string]string{
    "alice": "admin",
    "bob":   "user",
}

// Register business rules on startup
func init() {
    // Rule: Check if current user owns the specified item (fixed-arity, typed)
    rules.Register("owned_by", func(ctx context.Context, itemID *string, currentUser string) (bool, error) {
        if itemID == nil {
            return false, fmt.Errorf("owned_by: entity must be *string (item ID)")
        }

        owner, exists := itemOwners[*itemID]
        if !exists {
            return false, fmt.Errorf("owned_by: item %s not found", *itemID)
        }

        // Return false (validation failed) if ownership doesn't match
        return owner == currentUser, nil
    })

    // Rule: Check if user has required role (fixed-arity, typed)
    rules.Register("has_role", func(ctx context.Context, _ *string, user string, requiredRole string) (bool, error) {
        userRole, exists := userRoles[user]
        if !exists {
            return false, nil // User not found = validation failed
        }
        return userRole == requiredRole, nil
    })

    // Rule: Check if value is in allowed list (typed variadic)
    rules.Register("in_list", func(ctx context.Context, value *string, allowed ...string) (bool, error) {
        if value == nil {
            return false, fmt.Errorf("in_list: entity must be *string")
        }
        v := *value
        for _, a := range allowed {
            if a == v {
                return true, nil
            }
        }
        return false, nil // Not in allowed list
    })
}

// Request with rule-based validation
type UpdateItemRequest struct {
    Path struct {
        // ItemID must be owned by the current user
        ItemID string `gork:"itemId" validate:"required" rule:"owned_by($current_user)"`
        
        // Status must be one of the allowed values
        Status string `gork:"status" validate:"required" rule:"in_list('active', 'inactive', 'pending')"`
    }
    Body struct {
        // Name is required and must be owned by current user with admin role
        Name string `gork:"name" validate:"required" rule:"owned_by($current_user) && has_role($current_user, 'admin')"`
        
        // Category can reference other fields in complex expressions
        Category string `gork:"category" rule:"in_list('tech', 'business') || (.Name == 'special' && has_role($current_user, 'admin'))"`
    }
}

type UpdateItemResponse struct {
    Body struct {
        Success bool   `gork:"success"`
        Message string `gork:"message"`
    }
}

// Handler that automatically validates rules before execution
func UpdateItem(ctx context.Context, req UpdateItemRequest) (*UpdateItemResponse, error) {
    // If we reach here, all validation and rules have passed!
    return &UpdateItemResponse{
        Body: struct {
            Success bool   `gork:"success"`
            Message string `gork:"message"`
        }{
            Success: true,
            Message: fmt.Sprintf("Item %s updated successfully", req.Path.ItemID),
        },
    }, nil
}

func main() {
    mux := http.NewServeMux()
    router := stdlib.NewRouter(mux)
    
    // Context variables are injected via middleware
    router.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract user from JWT/session (simplified example)
            currentUser := r.Header.Get("X-Current-User")
            if currentUser == "" {
                currentUser = "anonymous"
            }
            
            // Make context variables available to rules
            ctx := rules.WithContextVars(r.Context(), rules.ContextVars{
                "current_user": currentUser,
                "user_role":    userRoles[currentUser],
                "request_time": "2024-01-15T10:00:00Z",
            })
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    })
    
    // Register route - rules are automatically applied during request validation
    router.Put("/items/{itemId}/status/{status}", UpdateItem, api.WithTags("items"))
    
    // Serve docs to see the generated OpenAPI spec
    router.DocsRoute("/docs/*")
    
    fmt.Println("Server running at http://localhost:8080")
    fmt.Println("Try: PUT /items/item-123/status/active with X-Current-User: alice")
    fmt.Println("Docs: http://localhost:8080/docs/")
    
    http.ListenAndServe(":8080", mux)
}
```

### Rule Expression Syntax

Rules support powerful expressions with field references, context variables, and boolean logic:

```go
type AdvancedRulesRequest struct {
    Path struct {
        ResourceID string `rule:"owned_by($current_user)"` // Context variable
        
        Action string `rule:"in_list('read', 'write', 'delete')"` // Literal arguments
    }
    Query struct {
        // Reference fields from other sections
        Permission string `rule:"has_permission($current_user, $.Path.Action)"` 
        
        // Complex boolean expressions
        Override bool `rule:"has_role($current_user, 'admin') || ($.Path.Action == 'read' && .Permission == 'public')"` 
    }
    Body struct {
        // Relative field references within same section
        Category string `rule:"in_list('tech', 'business')"`
        Tags     []string `rule:"valid_tags(.Category)"`  // Pass sibling field
    }
}
```

**Expression Features:**
- **Field References**: `$.Path.UserID` (absolute), `.Category` (relative to current section)
- **Context Variables**: `$current_user`, `$user_role`, `$request_time`
- **Literals**: `'string'`, `42`, `true`, `false`, `null`
- **Boolean Logic**: `&&` (and), `||` (or), `==` (equals)


### Manual Rule Application

For custom validation flows, you can apply rules manually:

```go
import "github.com/gork-labs/gork/pkg/rules"

func CustomHandler(ctx context.Context, req MyRequest) (*MyResponse, error) {
    // Apply rules manually
    if errs := rules.Apply(ctx, &req); len(errs) > 0 {
        return nil, fmt.Errorf("validation failed: %v", errs)
    }
    
    // Continue with business logic
    return &MyResponse{}, nil
}
```

### Key Benefits

- **🔐 Authorization**: Implement ownership and permission checks declaratively
- **📊 Business Logic**: Complex validation rules without cluttering handlers  
- **🎯 Reusable**: Register rules once, use across multiple endpoints
- **🌐 Context-Aware**: Access request context, user data, and session info
- **📝 Self-Documenting**: Rules are visible in struct definitions
- **⚡ Performance**: Compiled expressions, no runtime parsing overhead

## 📚 Libraries

### Core API Library
```bash
go get github.com/gork-labs/gork/pkg/api
```
Framework-agnostic API handlers with automatic OpenAPI metadata extraction and type-safe request/response handling.

### Webhooks
```bash
go get github.com/gork-labs/gork/pkg/webhooks/stripe
```
- **Typed Webhook Handling**: Define a provider handler and register event-specific functions with compile-time checked signatures.
- **Signature Verification**: Provider verifies signatures (Stripe via official SDK) and extracts provider payload + optional user metadata.
- **OpenAPI Extensions**: Webhook routes automatically include `x-webhook-provider` and `x-webhook-events` metadata in the generated spec.

Basic Stripe example:
```go
import (
  "net/http"
  "github.com/gork-labs/gork/pkg/api"
  stripepkg "github.com/gork-labs/gork/pkg/webhooks/stripe"
  "github.com/stripe/stripe-go/v76"
)

// User-defined metadata extracted from Stripe objects' Metadata field (optional)
type PaymentMetadata struct {
  UserID string `json:"user_id" validate:"required"`
}

func HandlePaymentSucceeded(ctx context.Context, pi *stripe.PaymentIntent, meta *PaymentMetadata) error {
  // process success; return error to signal failure
  return nil
}

func RegisterRoutes(mux *http.ServeMux) {
  r := stdlib.NewRouter(mux)

  r.Post(
    "/webhooks/stripe",
    api.WebhookHandlerFunc(
      stripepkg.NewHandler("whsec_example"), // verifies Stripe-Signature
      // Type parameters inferred from handler signature
      api.WithEventHandler("payment_intent.succeeded", HandlePaymentSucceeded),
    ),
    api.WithTags("webhooks", "stripe"),
  )
}
```

Notes:
- Provider returns standardized success/error JSON. Unhandled events return 200 with provider success response.
- Handlers have signature: `func(ctx context.Context, payload *ProviderType, meta *UserType) error`.
- Stripe provider maps common event families to concrete types (e.g., `*stripe.PaymentIntent`, `*stripe.Invoice`) and forwards `Metadata` as `meta`.

### Union Types
```bash  
go get github.com/gork-labs/gork/pkg/unions
```
Type-safe union types (`Union2`, `Union3`, `Union4`) with JSON marshaling and validation support for modeling API variants.

### Framework Adapters
Choose your web framework:
```bash
go get github.com/gork-labs/gork/pkg/adapters/gin      # Gin
go get github.com/gork-labs/gork/pkg/adapters/echo     # Echo  
go get github.com/gork-labs/gork/pkg/adapters/chi      # Chi
go get github.com/gork-labs/gork/pkg/adapters/fiber    # Fiber
go get github.com/gork-labs/gork/pkg/adapters/gorilla  # Gorilla Mux
go get github.com/gork-labs/gork/pkg/adapters/stdlib   # Standard library
```

## Development

This repository uses Go workspaces for local development. To get started:

```bash
# Clone the repository
git clone https://github.com/gork-labs/gork.git
cd gork

# Run tests for all modules
make test

# Build CLI tools
make build

# Generate coverage reports (requires 100% coverage)
make coverage

# Generate HTML coverage reports
make coverage-html

# Run linting
make lint

# Format code
make fmt

# Install dependencies
make deps

# Security vulnerability check
make vuln

# OpenAPI generation and validation
make openapi-gen
make openapi-validate

# Clean build artifacts
make clean
```

### Requirements

- Go 1.24 or higher
- Make (for using the Makefile)
- Go workspace support

### Project Structure

Each module in this monorepo:
- Has its own `go.mod` file
- Can be versioned independently
- Can be imported separately by users
- Shares common development tooling

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`make test`)
6. Submit a pull request

## Versioning

This monorepo uses independent versioning for each module:

- Module versions follow semantic versioning
- Tags use the format: `<module-path>/v<version>`
- Example: `pkg/unions/v1.0.0`, `openapi-gen/v2.1.0`

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🗺️ Roadmap

### ✅ Current Features
- **Convention Over Configuration**: Standardized request/response structure
- **Type-Safe Handlers**: Compile-time validation with strict signatures  
- **OpenAPI 3.1.0 Generation**: Automatic spec generation from Go source
- **Multi-Framework Support**: 6 popular Go web framework adapters
- **Union Types**: Type-safe variants with JSON marshaling
- **Static Analysis**: Custom linter for convention compliance
- **100% Test Coverage**: Quality-first development approach
- **Interactive Documentation**: Built-in docs serving
- **Webhook Utilities**: Typed event handlers, signature verification (Stripe), OpenAPI extensions

### 🚀 Coming Soon
- **⚡ Ahead-of-Time Compilation**: Eliminate runtime reflection for better performance
- **📝 Enhanced Documentation**: Improved OpenAPI spec generation
- **🌊 Event Streams**: WebSocket and SSE support  
- **🎯 Advanced Validation**: Build-time validation generation
- **📏 Simple Rule Engine**: Input validation with business rules (e.g., `rule:owned_by($current_user)`)
- **🔗 Variable-Length Unions**: User-defined union types with custom properties
  ```go
  type Events struct {
      Paid      *Paid
      Collected *Collected
      Cancelled *Cancelled
  }
  ```

## Support

- **Documentation**: See individual module READMEs
- **Issues**: [GitHub Issues](https://github.com/gork-labs/gork/issues)
- **Discussions**: [GitHub Discussions](https://github.com/gork-labs/gork/discussions)

## ⚡ Performance Note

Gork currently relies on reflection for type introspection and OpenAPI generation, prioritizing developer experience and rapid business development over raw performance. While this makes it one of the most business-development friendly frameworks available, we're working on **ahead-of-time compilation** for all reflection-dependent features to significantly improve baseline performance in future releases.

## Sponsors

- [MakeADir](https://makeadir.com) - No-code platform for building online directory websites

## Acknowledgments

This project builds upon excellent work from the Go community, including:
- [go-playground/validator](https://github.com/go-playground/validator)
- The Go standard library
- Various web frameworks (Gin, Echo, Chi, etc.)
