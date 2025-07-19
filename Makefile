# Root Makefile for gork monorepo
.PHONY: all test build clean lint list-modules coverage coverage-html deps verify fmt vuln test-%

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

# Lint all modules
lint:
	@for module in $(MODULES); do \
		echo "Linting $$module..."; \
		(cd $$module && golangci-lint run) || exit 1; \
	done

# Run tests with coverage and enforce thresholds
coverage:
	@for module in $(MODULES); do \
		if [ "$$module" = "examples" ]; then \
			echo "⏭️ Skipping coverage check for examples module"; \
			continue; \
		fi; \
		echo "Checking coverage for $$module (requires 100%)..."; \
		(cd $$module && go test ./... -coverprofile=coverage.out) || exit 1; \
		./scripts/check-coverage.sh $$module 100 || exit 1; \
	done

# Generate HTML coverage reports for all modules
coverage-html:
	@for module in $(MODULES); do \
		if [ "$$module" = "examples" ]; then \
			echo "⏭️ Skipping coverage report for examples module"; \
			continue; \
		fi; \
		echo "Generating HTML coverage for $$module..."; \
		(cd $$module && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html) || exit 1; \
		echo "Coverage report generated: $$module/coverage.html"; \
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
