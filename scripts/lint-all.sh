#!/bin/bash
# Lint all modules with failure collection

set -e

echo "Installing custom linter..."
if ! go install ./cmd/lintgork >/dev/null 2>&1; then
    echo "failed to install lintgork"
    exit 1
fi

# Read modules from go.work
MODULES=$(go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

failures=0
for module in $MODULES; do
    echo "Linting $module..."
    if ! (cd "$module" && golangci-lint run); then
        failures=$((failures+1))
    fi
done

if [ $failures -ne 0 ]; then
    echo "Linting completed with $failures failures."
    exit 1
else
    echo "Linting completed successfully."
fi
