#!/bin/bash
# Test a specific module by name
# Usage: ./test-module.sh pkg/adapters/chi

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <module_path>"
    echo "Example: $0 pkg/adapters/chi"
    exit 1
fi

target="$1"

# Read modules from go.work
MODULES=$(go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

found=0
for module in $MODULES; do
    if [ "$module" = "$target" ]; then
        echo "Testing $module..."
        (cd "$module" && go test ./... -v -cover)
        found=1
        break
    fi
done

if [ $found -eq 0 ]; then
    echo "Module '$target' not found in workspace"
    echo "Available modules:"
    for module in $MODULES; do
        echo "  $module"
    done
    exit 1
fi
