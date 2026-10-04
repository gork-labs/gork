# Root Makefile for gork monorepo
.PHONY: all test build clean lint coverage deps verify fmt vuln openapi-build openapi-gen openapi-validate openapi-swagger-validate openapi-lint

all: test build

test:
	go test ./... -v

# Checks that every package outside examples has 100% statement coverage.
coverage:
	@./scripts/check-coverage.sh

build:
	@./scripts/build-tools.sh

clean:
	rm -rf bin/
	go clean -cache -testcache

lint:
	golangci-lint run

# Update dependencies
deps:
	go mod tidy

# Verify dependencies
verify:
	go mod verify

# Format all Go code
fmt:
	gofumpt -w .

# Check for vulnerabilities
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Build the openapi-gen tool
openapi-build:
	@echo "Building gork CLI..."
	@go build -o bin/gork ./cmd/gork

# Generate OpenAPI specs for examples and testdata
openapi-gen: openapi-build
	@if [ -d "examples/cmd/openapi_export" ]; then \
		echo "Generating OpenAPI specs for examples..."; \
		./bin/gork openapi generate --build ./examples/cmd/openapi_export --source ./examples --output ./examples/openapi.json --title "API" --version "1.0.0"; \
		./bin/gork openapi generate --build ./examples/cmd/openapi_export --source ./examples --output ./examples/openapi.yaml --title "API" --version "1.0.0"; \
	fi

# Validate OpenAPI specs with Swagger validator API
openapi-swagger-validate:
	@echo "Validating OpenAPI specs with Swagger validator...";
	@validation_failed=0; \
	if [ -f "examples/openapi.json" ]; then \
		echo "Validating examples/openapi.json..."; \
		./scripts/validate-openapi.sh examples/openapi.json || validation_failed=1; \
	fi; \
	if [ -f "examples/openapi.yaml" ]; then \
		echo "Validating examples/openapi.yaml..."; \
		./scripts/validate-openapi.sh examples/openapi.yaml || validation_failed=1; \
	fi; \
	if [ $$validation_failed -eq 1 ]; then \
		echo "ERROR: One or more OpenAPI specs failed Swagger validation"; \
		exit 1; \
	fi; \
	echo "All OpenAPI specs passed Swagger validation!"

# Lint OpenAPI specs locally with Redocly CLI (via npx)
# Lints only examples/openapi.json and examples/openapi.yaml
openapi-lint: openapi-gen
	@echo "Linting OpenAPI specs with Redocly...";
	npx -y @redocly/cli lint --config .redocly.yaml examples/openapi.json
	npx -y @redocly/cli lint --config .redocly.yaml examples/openapi.yaml

# Validate that generated OpenAPI specs match committed ones and pass Swagger validation
openapi-validate: openapi-gen openapi-swagger-validate
	@echo "Comparing generated OpenAPI specs with committed ones..."
	@if [ -f "examples/openapi.json" ]; then \
		./bin/gork openapi generate --build ./examples/cmd/openapi_export --source ./examples --output ./examples/openapi-new.json --title "API" --version "1.0.0"; \
		if ! diff -q examples/openapi.json examples/openapi-new.json > /dev/null; then \
			echo "ERROR: examples/openapi.json is out of date!"; \
			echo "Run 'make openapi-gen' to regenerate."; \
			diff -u examples/openapi.json examples/openapi-new.json || true; \
			rm -f examples/openapi-new.json; \
			exit 1; \
		fi; \
		rm -f examples/openapi-new.json; \
	fi
	@echo "All OpenAPI specs are up to date!"
