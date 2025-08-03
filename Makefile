# Root Makefile for gork monorepo
.PHONY: all test build clean lint list-modules coverage coverage-html deps verify fmt vuln openapi-build openapi-gen openapi-validate openapi-swagger-validate

# Dynamically read modules from go.work (used only by list-modules and some remaining inline targets)
MODULES := $(shell go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

# Also define specific module groups for targeted operations
PKG_MODULES := $(filter pkg/%,$(MODULES))
TOOL_MODULES := $(filter-out pkg/% examples,$(MODULES))

all: test build

# Test specific module or all modules
# Usage: make test [module_path]
# Examples:
#   make test                    # test all modules
#   make test pkg/adapters/chi   # test specific module
test:
	@if [ -n "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		./scripts/test-module.sh "$(filter-out $@,$(MAKECMDGOALS))"; \
	else \
		./scripts/test-all.sh; \
	fi

# Coverage check for specific module or all modules
# Usage: make coverage [module_path]
# Examples:
#   make coverage                    # check coverage for all modules
#   make coverage pkg/adapters/chi   # check coverage for specific module
coverage:
	@if [ -n "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		./scripts/coverage-module.sh "$(filter-out $@,$(MAKECMDGOALS))"; \
	else \
		./scripts/coverage-all.sh; \
	fi

# Prevent make from treating module names as targets
%:
	@:

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
	@./scripts/build-tools.sh

clean:
	rm -rf bin/
	@for module in $(MODULES); do \
		(cd $$module && go clean -cache -testcache); \
	done

# Lint specific module or all modules
# Usage: make lint [module_path]
# Examples:
#   make lint                    # lint all modules
#   make lint pkg/adapters/chi   # lint specific module
lint:
	@if [ -n "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		./scripts/lint-module.sh "$(filter-out $@,$(MAKECMDGOALS))"; \
	else \
		./scripts/lint-all.sh; \
	fi

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
	@echo "Building gork CLI..."
	@(cd cmd/gork && go build -o ../../bin/gork .)

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
