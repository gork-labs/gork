# Root Makefile for gork monorepo
.PHONY: all test build clean lint lint-% list-modules coverage coverage-html deps verify fmt vuln test-% coverage-% openapi-build openapi-gen openapi-validate openapi-swagger-validate

# Dynamically read modules from go.work
MODULES := $(shell go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

# Also define specific module groups for targeted operations
PKG_MODULES := $(filter pkg/%,$(MODULES))
TOOL_MODULES := $(filter-out pkg/% examples,$(MODULES))

all: test build

test:
	@for module in $(MODULES); do \
		echo "Testing $$module..."; \
		(cd $$module && go test ./... -v) || exit 1; \
	done

# List all modules detected from go.work
list-modules:
	@if [ "$(FORMAT)" = "json" ]; then \
		go work edit -json | jq -c '[.Use[].DiskPath | sub("^\\./"; "")]'; \
	else \
		echo "All modules:"; \
		for module in $(MODULES); do echo "  - $$module"; done; \
		echo ""; \
		echo "Package modules:"; \
		for module in $(PKG_MODULES); do echo "  - $$module"; done; \
		echo ""; \
		echo "Tool modules:"; \
		for module in $(TOOL_MODULES); do echo "  - $$module"; done; \
	fi

build:
	@for module in $(TOOL_MODULES); do \
		echo "Building $$module..."; \
		if [ -d "$$module/cmd" ]; then \
			for cmd in $$module/cmd/*; do \
				if [ -d "$$cmd" ]; then \
					cmdname=$$(basename $$cmd); \
					echo "  Building $$cmdname..."; \
					outpath=$$(echo "$$module" | sed 's|[^/]*|..|g')/bin/$$cmdname; \
					(cd $$module && go build -o $$outpath ./cmd/$$cmdname) || exit 1; \
				fi; \
			done; \
		fi; \
	done

clean:
	rm -rf bin/
	@for module in $(MODULES); do \
		(cd $$module && go clean -cache -testcache); \
	done

# Dynamic module-specific test targets
# Usage: make test-unions, make test-api, make test-openapi-gen, etc.
test-%:
	@target="$*"; \
	found=0; \
	for module in $(MODULES); do \
		if echo "$$module" | grep -qE "(^|/)$$target$$"; then \
			echo "Testing $$module..."; \
			cd $$module && go test ./... -v -cover; \
			found=1; \
			break; \
		fi; \
	done; \
	if [ $$found -eq 0 ]; then \
		echo "Module '$$target' not found in workspace"; \
		echo "Available modules: $(MODULES)"; \
		exit 1; \
	fi

# Dynamic module-specific coverage targets
# Usage: make coverage-unions, make coverage-api, make coverage-openapi-gen, etc.
coverage-%:
	@target="$*"; \
	found=0; \
	for module in $(MODULES); do \
		if echo "$$module" | grep -qE "(^|/)$$target$$"; then \
			./scripts/check-coverage.sh $$module 100 || exit 1; \
			found=1; \
			break; \
		fi; \
	done; \
	if [ $$found -eq 0 ]; then \
		echo "Module '$$target' not found in workspace"; \
		echo "Available modules: $(MODULES)"; \
		exit 1; \
	fi

# Lint all modules
lint:
	@for module in $(MODULES); do \
		echo "Linting $$module..."; \
		(cd $$module && golangci-lint run) || exit 1; \
	done

# Dynamic module-specific lint targets
# Usage: make lint-unions, make lint-api, make lint-openapi-gen, etc.
lint-%:
	@target="$*"; \
	found=0; \
	for module in $(MODULES); do \
		if echo "$$module" | grep -qE "(^|/)$$target$$"; then \
			echo "Linting $$module..."; \
			cd $$module && golangci-lint run; \
			found=1; \
			break; \
		fi; \
	done; \
	if [ $$found -eq 0 ]; then \
		echo "Module '$$target' not found in workspace"; \
		echo "Available modules: $(MODULES)"; \
		exit 1; \
	fi

# Run tests with coverage and enforce thresholds
coverage:
	@for module in $(MODULES); do \
		./scripts/check-coverage.sh $$module 100 || exit 1; \
	done

# Generate HTML coverage reports for all modules
coverage-html:
	@for module in $(MODULES); do \
		./scripts/check-coverage.sh $$module 100 --html || exit 1; \
	done

# Update dependencies
deps:
	@for module in $(MODULES); do \
		echo "Updating dependencies for $$module..."; \
		(cd $$module && go mod tidy) || exit 1; \
	done

# Verify all modules
verify:
	@for module in $(MODULES); do \
		echo "Verifying $$module..."; \
		(cd $$module && go mod verify) || exit 1; \
	done

# Format all Go code
fmt:
	@for module in $(MODULES); do \
		echo "Formatting $$module..."; \
		(cd $$module && go fmt ./...) || exit 1; \
	done

# Check for vulnerabilities
vuln:
	@for module in $(MODULES); do \
		echo "Checking vulnerabilities in $$module..."; \
		(cd $$module && go run golang.org/x/vuln/cmd/govulncheck@latest ./...) || exit 1; \
	done

# Build the openapi-gen tool
openapi-build:
	@echo "Building openapi-gen tool..."
	@(cd tools/openapi-gen && go build -o ../../bin/openapi-gen ./cmd/openapi-gen)

# Generate OpenAPI specs for examples and testdata
openapi-gen: openapi-build
	@if [ -d "examples" ] && [ -f "examples/routes.go" ]; then \
		echo "Generating OpenAPI specs for examples..."; \
		./bin/openapi-gen -i examples -r examples/routes.go -o examples/openapi.json; \
		./bin/openapi-gen -i examples -r examples/routes.go -o examples/openapi.yaml -f yaml; \
	fi
	@if [ -d "tools/openapi-gen/testdata" ] && [ -f "tools/openapi-gen/testdata/routes.go" ]; then \
		echo "Generating OpenAPI specs for testdata..."; \
		./bin/openapi-gen -i tools/openapi-gen/testdata -r tools/openapi-gen/testdata/routes.go -o tools/openapi-gen/testdata/openapi.json; \
		./bin/openapi-gen -i tools/openapi-gen/testdata -r tools/openapi-gen/testdata/routes.go -o tools/openapi-gen/testdata/openapi.yaml -f yaml; \
	fi

# Validate OpenAPI specs with Swagger validator API
openapi-swagger-validate:
	@echo "Validating OpenAPI specs with Swagger validator..."
	@validation_failed=0; \
	if [ -f "examples/openapi.json" ]; then \
		echo "Validating examples/openapi.json..."; \
		./scripts/validate-openapi.sh examples/openapi.json || validation_failed=1; \
	fi; \
	if [ -f "examples/openapi.yaml" ]; then \
		echo "Validating examples/openapi.yaml..."; \
		./scripts/validate-openapi.sh examples/openapi.yaml || validation_failed=1; \
	fi; \
	if [ -f "tools/openapi-gen/testdata/openapi.json" ]; then \
		echo "Validating testdata/openapi.json..."; \
		./scripts/validate-openapi.sh tools/openapi-gen/testdata/openapi.json || validation_failed=1; \
	fi; \
	if [ -f "tools/openapi-gen/testdata/openapi.yaml" ]; then \
		echo "Validating testdata/openapi.yaml..."; \
		./scripts/validate-openapi.sh tools/openapi-gen/testdata/openapi.yaml || validation_failed=1; \
	fi; \
	if [ $$validation_failed -eq 1 ]; then \
		echo "ERROR: One or more OpenAPI specs failed Swagger validation"; \
		exit 1; \
	fi; \
	echo "All OpenAPI specs passed Swagger validation!"

# Validate that generated OpenAPI specs match committed ones and pass Swagger validation
openapi-validate: openapi-gen openapi-swagger-validate
	@echo "Comparing generated OpenAPI specs with committed ones..."
	@if [ -f "examples/openapi.json" ]; then \
		./bin/openapi-gen -i examples -r examples/routes.go -o examples/openapi-new.json; \
		if ! diff -q examples/openapi.json examples/openapi-new.json > /dev/null; then \
			echo "ERROR: examples/openapi.json is out of date!"; \
			echo "Run 'make openapi-gen' to regenerate."; \
			diff -u examples/openapi.json examples/openapi-new.json || true; \
			rm -f examples/openapi-new.json; \
			exit 1; \
		fi; \
		rm -f examples/openapi-new.json; \
	fi
	@if [ -f "examples/openapi.yaml" ]; then \
		./bin/openapi-gen -i examples -r examples/routes.go -o examples/openapi-new.yaml -f yaml; \
		if ! diff -q examples/openapi.yaml examples/openapi-new.yaml > /dev/null; then \
			echo "ERROR: examples/openapi.yaml is out of date!"; \
			echo "Run 'make openapi-gen' to regenerate."; \
			diff -u examples/openapi.yaml examples/openapi-new.yaml || true; \
			rm -f examples/openapi-new.yaml; \
			exit 1; \
		fi; \
		rm -f examples/openapi-new.yaml; \
	fi
	@if [ -f "tools/openapi-gen/testdata/openapi.json" ]; then \
		./bin/openapi-gen -i tools/openapi-gen/testdata -r tools/openapi-gen/testdata/routes.go -o tools/openapi-gen/testdata/openapi-new.json; \
		if ! diff -q tools/openapi-gen/testdata/openapi.json tools/openapi-gen/testdata/openapi-new.json > /dev/null; then \
			echo "ERROR: testdata/openapi.json is out of date!"; \
			echo "Run 'make openapi-gen' to regenerate."; \
			diff -u tools/openapi-gen/testdata/openapi.json tools/openapi-gen/testdata/openapi-new.json || true; \
			rm -f tools/openapi-gen/testdata/openapi-new.json; \
			exit 1; \
		fi; \
		rm -f tools/openapi-gen/testdata/openapi-new.json; \
	fi
	@if [ -f "tools/openapi-gen/testdata/openapi.yaml" ]; then \
		./bin/openapi-gen -i tools/openapi-gen/testdata -r tools/openapi-gen/testdata/routes.go -o tools/openapi-gen/testdata/openapi-new.yaml -f yaml; \
		if ! diff -q tools/openapi-gen/testdata/openapi.yaml tools/openapi-gen/testdata/openapi-new.yaml > /dev/null; then \
			echo "ERROR: testdata/openapi.yaml is out of date!"; \
			echo "Run 'make openapi-gen' to regenerate."; \
			diff -u tools/openapi-gen/testdata/openapi.yaml tools/openapi-gen/testdata/openapi-new.yaml || true; \
			rm -f tools/openapi-gen/testdata/openapi-new.yaml; \
			exit 1; \
		fi; \
		rm -f tools/openapi-gen/testdata/openapi-new.yaml; \
	fi
	@echo "All OpenAPI specs are up to date!"
