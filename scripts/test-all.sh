#!/bin/bash
# Test all modules with failure collection

set -e

# Read modules from go.work
MODULES=$(go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

failures=0
for module in $MODULES; do
    echo "Testing $module..."
    if ! (cd "$module" && go test ./... -v); then
        failures=$((failures+1))
    fi
done

if [ $failures -ne 0 ]; then
    echo "Testing completed with $failures failures."
    exit 1
else
    echo "Testing completed successfully."
fi
