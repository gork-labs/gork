# Gork - Go API Development Toolkit

[![CI](https://github.com/gork-labs/gork/workflows/CI/badge.svg)](https://github.com/gork-labs/gork/actions)
[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg)](https://codecov.io/gh/gork-labs/gork)
[![Go Report Card](https://goreportcard.com/badge/github.com/gork-labs/gork)](https://goreportcard.com/report/github.com/gork-labs/gork)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Gork is a Go API development toolkit that provides type-safe HTTP handlers, automatic OpenAPI 3.1.0 generation, and union types. The toolkit includes multiple framework adapters and automatic API documentation generation from Go source code using go-playground/validator tags.

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

## CLI Tools

### gork

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg?flag=cmd%2Fgork)](https://codecov.io/gh/gork-labs/gork/tree/main/cmd/gork)

A CLI tool for OpenAPI 3.1.0 specification generation that extracts API documentation from Go source code using struct tags and type information.

**Installation:**
```bash
go install github.com/gork-labs/gork/cmd/gork@latest
```

**Features:**
- Automatic OpenAPI spec generation from Go code
- Support for go-playground/validator tags
- Union type support with discriminators  
- Multiple web framework support (Gin, Echo, Chi, Gorilla Mux, Fiber, standard library)
- JSON and YAML output formats
- Build-time and runtime spec generation
- Custom validator support

**Usage:**
```bash
# Basic usage with build-time generation
gork openapi generate --build ./examples/cmd/openapi_export --source ./examples --output openapi.json

# Runtime generation from source only
gork openapi generate --source ./handlers --output openapi.json

# YAML output with custom metadata
gork openapi generate --source ./pkg --output spec.yaml --format yaml --title "My API" --version "2.0.0"

# Using config file
gork openapi generate --config .gork.yml
```

### lintgork

A custom linter for struct validation and OpenAPI compliance that ensures your Go structs follow best practices for API generation.

**Installation:**
```bash
go install github.com/gork-labs/gork/cmd/lintgork@latest
```

**Features:**
- Validates struct tags for OpenAPI compliance
- Checks path parameter consistency
- Ensures proper discriminator usage
- Integrates with golangci-lint

### pkg/unions

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg?flag=pkg%2Funions)](https://codecov.io/gh/gork-labs/gork/tree/main/pkg/unions)

Type-safe union types for Go with JSON marshaling/unmarshaling support.

**Installation:**
```bash
go get github.com/gork-labs/gork/pkg/unions
```

**Features:**
- Union2, Union3, and Union4 types
- Type-safe access methods
- JSON serialization with discriminators
- Validation support

[Read more →](./pkg/unions/README.md)

## Libraries

### pkg/api

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg?flag=pkg%2Fapi)](https://codecov.io/gh/gork-labs/gork/tree/main/pkg/api)

HTTP handler adapter for building type-safe APIs with automatic OpenAPI metadata extraction.

**Installation:**
```bash
go get github.com/gork-labs/gork/pkg/api
```

**Features:**
- Type-safe request/response handling
- Automatic error responses
- OpenAPI metadata extraction
- Context propagation
- Framework-agnostic design

[Read more →](./pkg/api/README.md)

### pkg/adapters

Framework-specific adapters that integrate the core API functionality with popular Go web frameworks:

- **stdlib** - Standard library (`http.ServeMux`) adapter
- **gin** - Gin framework adapter
- **echo** - Echo framework adapter  
- **chi** - Chi router adapter
- **fiber** - Fiber framework adapter
- **gorilla** - Gorilla Mux adapter

Each adapter provides:
- Seamless integration with framework routing
- Parameter extraction from framework contexts
- Path parameter conversion
- Middleware support

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

## Roadmap

### Current (Implemented)
- ✅ OpenAPI 3.1.0 generator with full validator tag support
- ✅ Type-safe union types (Union2, Union3, Union4)
- ✅ HTTP API adapter with metadata extraction
- ✅ Multi-framework route detection (Gin, Echo, Chi, Gorilla Mux, Fiber, stdlib)
- ✅ Build-time and runtime spec generation
- ✅ JSON/YAML output formats
- ✅ Custom linter for struct validation and OpenAPI compliance
- ✅ 100% test coverage enforcement
- ✅ Defensive slice copying to prevent aliasing issues
- ✅ Comprehensive error wrapping for better debugging

### Planned
- 🚧 Webhook signature verification utilities
- 🚧 OpenAPI documentation serving middleware
- 🚧 Enhanced union type discriminator support

## Support

- **Documentation**: See individual module READMEs
- **Issues**: [GitHub Issues](https://github.com/gork-labs/gork/issues)
- **Discussions**: [GitHub Discussions](https://github.com/gork-labs/gork/discussions)

## Sponsors

- [MakeADir](https://makeadir.com) - No-code platform for building online directory websites

## Acknowledgments

This project builds upon excellent work from the Go community, including:
- [go-playground/validator](https://github.com/go-playground/validator)
- The Go standard library
- Various web frameworks (Gin, Echo, Chi, etc.)
