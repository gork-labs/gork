.PHONY: all clean test test-unit test-integration test-e2e test-bench test-race test-coverage test-verbose clean-test build

# Build the CLI binary
build:
	@echo "Building openapi-gen..."
	@go build -o bin/openapi-gen ./cmd/openapi-gen
	@echo "Binary built at bin/openapi-gen"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run all tests with coverage
test:
	@echo "Running all tests with coverage..."
	@go test ./... -v -race -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Tests completed. Coverage report generated at coverage.html"

# Run only unit tests
test-unit:
	@echo "Running unit tests..."
	@go test ./internal/generator -v -run "Test[^E2E]" -race

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test ./internal/generator -v -run "TestIntegration|TestFullGeneration" -race

# Run end-to-end tests
test-e2e:
	@echo "Running E2E tests..."
	@go test ./cmd/openapi-gen -v -run "TestE2E" -race

# Run benchmark tests
test-bench:
	@echo "Running benchmark tests..."
	@go test ./internal/generator -bench=. -benchmem -run=^Benchmark

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	@go test ./... -race -short

# Generate test coverage report
test-coverage:
	@echo "Generating test coverage report..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total
	@echo "Coverage report generated at coverage.html"

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test ./... -v -race

# Run tests for specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg PKG=package_name"; exit 1; fi
	@echo "Running tests for package $(PKG)..."
	@go test ./$(PKG) -v -race

# Run specific test
test-run:
	@if [ -z "$(TEST)" ]; then echo "Usage: make test-run TEST=TestName"; exit 1; fi
	@echo "Running test $(TEST)..."
	@go test ./... -v -run "$(TEST)" -race

# Lint the code
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@golangci-lint run

# Format the code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w . 2>/dev/null || echo "goimports not available, skipping import formatting"

# Vet the code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Run all quality checks
check: fmt vet lint test
	@echo "All quality checks passed!"

# Generate test data
generate-testdata:
	@echo "Generating test data..."
	@mkdir -p testdata/generated
	@go run ./cmd/openapi-gen -input ./testdata/simple_api -output ./testdata/generated/simple_api.json -title "Simple API" -version "1.0.0"
	@go run ./cmd/openapi-gen -input ./testdata/complex_api -output ./testdata/generated/complex_api.json -title "Complex API" -version "1.0.0"
	@echo "Test data generated in testdata/generated/"

# Generate example - runs OpenAPI generation
generate-example:
	@echo "Generating OpenAPI spec for examples..."
	@go run ./cmd/openapi-gen -i examples -r examples/routes.go -o examples/openapi.json -t "Example API" -v "1.0.0"
	@echo "Example generation complete!"

# Clean test artifacts
clean-test:
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@rm -rf testdata/generated
	@find . -name "*.test" -delete
	@find . -name "*.prof" -delete

# Clean all build artifacts
clean: clean-test
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f openapi-gen

# Run tests in CI mode (strict)
test-ci:
	@echo "Running tests in CI mode..."
	@go test ./... -race -coverprofile=coverage.out -covermode=atomic -timeout=10m
	@go tool cover -func=coverage.out

# Performance testing
perf:
	@echo "Running performance tests..."
	@go test ./internal/generator -bench=. -benchmem -count=3 -run=^Benchmark | tee benchmark.txt

# Memory profile
profile-mem:
	@echo "Running memory profiling..."
	@go test ./internal/generator -bench=BenchmarkFullGeneration -memprofile=mem.prof -run=^Benchmark
	@go tool pprof mem.prof

# CPU profile  
profile-cpu:
	@echo "Running CPU profiling..."
	@go test ./internal/generator -bench=BenchmarkFullGeneration -cpuprofile=cpu.prof -run=^Benchmark
	@go tool pprof cpu.prof

# Update golden files (use with caution)
update-golden:
	@echo "Updating golden files..."
	@echo "WARNING: This will overwrite existing golden files!"
	@read -p "Are you sure? [y/N] " -r; if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		go run ./cmd/openapi-gen -input ./testdata/simple_api -output ./testdata/golden/simple_api_expected.json -title "Test API" -version "1.0.0"; \
		echo "Golden files updated!"; \
	else \
		echo "Cancelled."; \
	fi

# Quick test (no race detection, faster for development)
test-quick:
	@echo "Running quick tests..."
	@go test ./... -short

# Test with timeout
test-timeout:
	@echo "Running tests with timeout..."
	@go test ./... -timeout=5m -race

# Generate mocks (if using mockgen)
generate-mocks:
	@echo "Generating mocks..."
	@which mockgen > /dev/null || (echo "mockgen not installed. Install with: go install github.com/golang/mock/mockgen@latest" && exit 1)
	@go generate ./...

# Help target
help:
	@echo "Available targets:"
	@echo "  build              - Build the CLI binary"
	@echo "  deps               - Install dependencies"
	@echo "  test               - Run all tests with coverage"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests"
	@echo "  test-e2e           - Run end-to-end tests"
	@echo "  test-bench         - Run benchmark tests"
	@echo "  test-race          - Run tests with race detection"
	@echo "  test-coverage      - Generate test coverage report"
	@echo "  test-verbose       - Run tests with verbose output"
	@echo "  test-pkg PKG=name  - Run tests for specific package"
	@echo "  test-run TEST=name - Run specific test"
	@echo "  test-quick         - Run quick tests (no race detection)"
	@echo "  test-timeout       - Run tests with timeout"
	@echo "  test-ci            - Run tests in CI mode"
	@echo "  lint               - Run linters"
	@echo "  fmt                - Format code"
	@echo "  vet                - Vet code"
	@echo "  check              - Run all quality checks"
	@echo "  perf               - Run performance tests"
	@echo "  profile-mem        - Run memory profiling"
	@echo "  profile-cpu        - Run CPU profiling"
	@echo "  generate-testdata  - Generate test data"
	@echo "  generate-example   - Generate OpenAPI spec for examples"
	@echo "  update-golden      - Update golden files"
	@echo "  clean              - Clean all artifacts"
	@echo "  clean-test         - Clean test artifacts"
	@echo "  help               - Show this help"
