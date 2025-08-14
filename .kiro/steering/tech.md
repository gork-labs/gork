# Technology Stack

## Language & Runtime
- **Go 1.24+** - Primary language with workspace support
- **Go Workspaces** - Monorepo management with independent module versioning

## Build System
- **Make** - Primary build automation via `Makefile`
- **Go Modules** - Dependency management with `go.mod` per module
- **Shell Scripts** - Located in `scripts/` for common operations

## Key Dependencies
- **go-playground/validator** - Struct validation with OpenAPI constraint mapping
- **Stripe Go SDK** - Webhook handling examples
- **golangci-lint** - Code quality and linting

## Common Commands

### Development Workflow
```bash
# Test all modules
make test

# Test specific module
make test pkg/api

# Build CLI tools (gork, lintgork)
make build

# Run linting on all modules
make lint

# Check test coverage (requires 100%)
make coverage

# Generate HTML coverage reports
make coverage-html

# Format all code
make fmt

# Update dependencies
make deps

# Security vulnerability check
make vuln
```

### OpenAPI Operations
```bash
# Generate OpenAPI specs for examples
make openapi-gen

# Validate generated specs match committed ones
make openapi-validate

# Build gork CLI tool
make openapi-build
```

### Module-Specific Operations
```bash
# Test specific module
./scripts/test-module.sh pkg/api

# Check coverage for specific module
./scripts/coverage-module.sh pkg/api

# Lint specific module
./scripts/lint-module.sh pkg/api
```

## Code Quality Standards
- **100% test coverage** required for all modules
- **golangci-lint** with strict configuration (`.golangci.yml`)
- **Custom linter** (`lintgork`) for convention compliance
- **Go formatting** with `gofmt`, `goimports`, `gofumpt`

## Framework Adapters
The project supports multiple Go web frameworks through adapters:
- `pkg/adapters/stdlib` - Standard library `http.ServeMux`
- `pkg/adapters/gin` - Gin framework
- `pkg/adapters/echo` - Echo framework
- `pkg/adapters/chi` - Chi router
- `pkg/adapters/fiber` - Fiber framework
- `pkg/adapters/gorilla` - Gorilla Mux

## CLI Tools
- **gork** - OpenAPI generation and validation
- **lintgork** - Convention compliance checking