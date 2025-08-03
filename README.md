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

## 📚 Libraries

### Core API Library
```bash
go get github.com/gork-labs/gork/pkg/api
```
Framework-agnostic API handlers with automatic OpenAPI metadata extraction and type-safe request/response handling.

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

### 🚀 Coming Soon
- **⚡ Ahead-of-Time Compilation**: Eliminate runtime reflection for better performance
- **📝 Enhanced Documentation**: Improved OpenAPI spec generation
- **🔒 Webhook Utilities**: Signature verification and event handling
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
