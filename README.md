# Gork - Go Development Tools Monorepo

[![CI](https://github.com/gork-labs/gork/workflows/CI/badge.svg)](https://github.com/gork-labs/gork/actions)
[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg)](https://codecov.io/gh/gork-labs/gork)
[![Go Report Card](https://goreportcard.com/badge/github.com/gork-labs/gork)](https://goreportcard.com/report/github.com/gork-labs/gork)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Gork is a collection of Go development tools designed to enhance productivity and code quality. This monorepo contains multiple tools and libraries that work together to provide a comprehensive development experience.

## Repository Structure

```
gork/
├── pkg/
│   ├── api/           # HTTP API adapter utilities
│   ├── unions/        # Type-safe union types for Go
│   └── webhooks/      # (Future) Webhook handling utilities
├── tools/
│   └── openapi-gen/   # OpenAPI 3.1.0 code generator
├── examples/          # Example projects
└── bin/               # Compiled binaries
```

## Modules

### openapi-gen

[![codecov](https://codecov.io/gh/gork-labs/gork/branch/main/graph/badge.svg?flag=tools%2Fopenapi-gen)](https://codecov.io/gh/gork-labs/gork/tree/main/tools/openapi-gen)

An OpenAPI 3.1.0 specification generator that extracts API documentation from Go source code using struct tags and type information.

**Installation:**
```bash
go install github.com/gork-labs/gork/tools/openapi-gen/cmd/openapi-gen@latest
```

**Features:**
- Automatic OpenAPI spec generation from Go code
- Support for go-playground/validator tags
- Union type support with discriminators
- Multiple web framework support (Gin, Echo, Chi, etc.)

[Read more →](./tools/openapi-gen/README.md)

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

[Read more →](./pkg/api/README.md)

## Development

This repository uses Go workspaces for local development. To get started:

```bash
# Clone the repository
git clone https://github.com/gork-labs/gork.git
cd gork

# Run tests for all modules
make test

# Build all tools
make build

# Run specific module tests
make test-openapi
make test-unions
make test-api
```

### Requirements

- Go 1.24 or higher
- Make (for using the Makefile)

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

### Current
- ✅ OpenAPI generator with union type support
- ✅ Type-safe union types
- ✅ HTTP API adapter

### Planned
- 🚧 Webhook signature verification utilities
- 🚧 gRPC to HTTP gateway generator
- 🚧 Schema validation tools
- 🚧 API linting tools

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
