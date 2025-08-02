#!/bin/bash

# Coverage enforcement script for CI/CD pipelines
# Usage: ./scripts/check-coverage.sh [module_path] [threshold] [--html]

set -e

MODULE_PATH="${1:-.}"
THRESHOLD="${2:-100}"
GENERATE_HTML=false

# Check for --html flag
if [[ "$3" == "--html" ]]; then
    GENERATE_HTML=true
fi

COVERAGE_FILE="${MODULE_PATH}/coverage.out"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Skip coverage check only for examples module
if [[ "$MODULE_PATH" == "examples" || "$MODULE_PATH" == "./examples" ]]; then
    echo -e "${YELLOW}⏭️ Skipping coverage check for examples module${NC}"
    exit 0
fi

# Set coverage thresholds per module - all modules now require 100% coverage
case "${MODULE_PATH}" in
    "pkg/api" | "./pkg/api")
        THRESHOLD="${2:-100}"
        ;;
    "pkg/adapters/fiber" | "./pkg/adapters/fiber")
        THRESHOLD="${2:-100}"
        ;;
    "internal/lintgork" | "./internal/lintgork")
        THRESHOLD="${2:-100}"
        ;;
    pkg/adapters/* | ./pkg/adapters/*)
        THRESHOLD="${2:-100}"
        ;;
    pkg/unions* | ./pkg/unions*)
        THRESHOLD="${2:-100}"
        ;;
    *)
        THRESHOLD="${2:-100}"
        ;;
esac

echo -e "${BLUE}🔍 Checking coverage for module: ${MODULE_PATH}${NC}"
echo -e "${BLUE}📊 Required threshold: ${THRESHOLD}%${NC}"

# Run tests with coverage
cd "$MODULE_PATH"
if [[ "$MODULE_PATH" == "tools/openapi-gen" || "$MODULE_PATH" == "./tools/openapi-gen" ]]; then
    echo -e "${BLUE}📦 Excluding cmd/ directory from tests and coverage${NC}"
    TEST_PACKAGES=$(go list ./... | grep -v '/cmd/')
    go test $TEST_PACKAGES -v -race -coverprofile=coverage.out || exit 1
else
    go test ./... -v -race -coverprofile=coverage.out || exit 1
fi

# Check if coverage file exists (we're now in the module directory)
if [ ! -f "coverage.out" ]; then
    echo -e "${RED}❌ Coverage file not found: coverage.out${NC}"
    echo -e "${YELLOW}💡 Run 'go test ./... -coverprofile=coverage.out' first${NC}"
    exit 1
fi

# Get coverage percentage
COVERAGE=$(go tool cover -func="coverage.out" | grep total | awk '{print $3}' | sed 's/%//')

if [ -z "$COVERAGE" ]; then
    echo -e "${RED}❌ Could not parse coverage from coverage.out${NC}"
    exit 1
fi

echo -e "${BLUE}📈 Current coverage: ${COVERAGE}%${NC}"

# Check coverage against threshold
if (( $(echo "$COVERAGE >= $THRESHOLD" | bc -l) )); then
    if (( $(echo "$COVERAGE == 100" | bc -l) )); then
        echo -e "${GREEN}🎉 Perfect coverage! ${COVERAGE}% - All code is tested!${NC}"
    elif (( $(echo "$COVERAGE >= 95" | bc -l) )); then
        echo -e "${GREEN}� Excellent coverage! ${COVERAGE}% (≥${THRESHOLD}%)${NC}"
    elif (( $(echo "$COVERAGE >= 90" | bc -l) )); then
        echo -e "${GREEN}✅ Great coverage! ${COVERAGE}% (≥${THRESHOLD}%)${NC}"
    else
        echo -e "${GREEN}✅ Good coverage! ${COVERAGE}% (≥${THRESHOLD}%)${NC}"
    fi

    # Generate coverage badge data
    if [ "$COVERAGE" ]; then
        if (( $(echo "$COVERAGE == 100" | bc -l) )); then
            COLOR="brightgreen"
        elif (( $(echo "$COVERAGE >= 95" | bc -l) )); then
            COLOR="green"
        elif (( $(echo "$COVERAGE >= 90" | bc -l) )); then
            COLOR="yellowgreen"
        elif (( $(echo "$COVERAGE >= 80" | bc -l) )); then
            COLOR="yellow"
        else
            COLOR="orange"
        fi

        if [ -n "$GITHUB_ENV" ]; then
            echo "COVERAGE_BADGE=https://img.shields.io/badge/coverage-${COVERAGE}%25-${COLOR}" >> "$GITHUB_ENV" 2>/dev/null || true
        fi
    fi

    # Generate HTML report if requested
    if [ "$GENERATE_HTML" = true ]; then
        go tool cover -html="coverage.out" -o coverage.html
        echo -e "${GREEN}📄 Coverage report generated: ${MODULE_PATH}/coverage.html${NC}"
    fi

    exit 0
else
    echo -e "${RED}❌ Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%${NC}"

    # Show detailed coverage breakdown for missing coverage
    echo -e "${YELLOW}📋 Functions with missing coverage:${NC}"
    go tool cover -func="coverage.out" | grep -v "100.0%" | head -20

    echo -e "${YELLOW}📋 Summary by file:${NC}"
    go tool cover -func="coverage.out" | grep -E "\.(go):" | sort -k3 -nr | head -10

    if [ -t 1 ]; then  # Only if running in terminal
        echo -e "${YELLOW}💡 To achieve 100% coverage:${NC}"
        echo -e "   1. Add tests for all uncovered functions"
        echo -e "   2. Test all error paths and edge cases"
        echo -e "   3. Test all conditional branches (if/else/switch)"
        echo -e "   4. Add integration tests for complex workflows"
        echo -e "   5. Generate HTML report: go tool cover -html=coverage.out -o coverage.html"
        echo -e "${BLUE}🎯 Goal: Every line of code should be tested!${NC}"
    fi

    # Generate HTML report even on failure if requested
    if [ "$GENERATE_HTML" = true ]; then
        go tool cover -html="coverage.out" -o coverage.html
        echo -e "${YELLOW}📄 Coverage report generated: ${MODULE_PATH}/coverage.html${NC}"
    fi

    exit 1
fi
