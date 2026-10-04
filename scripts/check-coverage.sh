#!/bin/bash

# Coverage enforcement script for CI/CD pipelines
# Usage: ./scripts/check-coverage.sh
# Every package outside examples must have 100% statement coverage.
# HTML coverage reports are always generated automatically

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Checking coverage for all packages outside examples${NC}"

TEST_PACKAGES=$(go list ./... | grep -v '/examples')
go test $TEST_PACKAGES -v -race -coverprofile=coverage.out || exit 1

if [ ! -f "coverage.out" ]; then
    echo -e "${RED}❌ Coverage file not found: coverage.out${NC}"
    exit 1
fi

COVERAGE=$(go tool cover -func="coverage.out" | grep total | awk '{print $3}')
echo -e "${BLUE}📈 Current coverage: ${COVERAGE}${NC}"

go tool cover -html="coverage.out" -o coverage.html

# The rounded total can show 100.0% with uncovered statements, so check each block.
# A profile line is "file:startLine.startCol,endLine.endCol numStatements hitCount".
UNCOVERED=$(awk 'NR > 1 { stmts[$1] = $2; hits[$1] += $3 } END { for (b in stmts) if (stmts[b] > 0 && hits[b] == 0) print b }' coverage.out | sort)

if [ -z "$UNCOVERED" ]; then
    echo -e "${GREEN}🎉 Perfect coverage! All code is tested!${NC}" >&2
    echo -e "${GREEN}📄 Coverage report generated: coverage.html${NC}" >&2
    exit 0
fi

echo -e "${RED}❌ Some statements have no test coverage${NC}"

echo -e "${YELLOW}📋 Packages below 100%:${NC}"
echo "$UNCOVERED" | sed 's|/[^/]*\.go:.*||' | sort -u | sed 's/^/  - /'

echo -e "${YELLOW}📋 Uncovered blocks:${NC}"
echo "$UNCOVERED" | head -50

echo -e "${YELLOW}📋 Functions with missing coverage:${NC}"
go tool cover -func="coverage.out" | grep -v "100.0%" | head -20

echo -e "${YELLOW}📄 Coverage report generated: coverage.html${NC}"

exit 1
