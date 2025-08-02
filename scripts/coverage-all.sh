#!/bin/bash
# Coverage check for all modules with failure collection

set -e

# Read modules from go.work
MODULES=$(go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

failures=0
for module in $MODULES; do
    echo "Checking coverage for $module..."
    # Use per-module thresholds defined in check-coverage.sh
    if ! ./scripts/check-coverage.sh "$module"; then
        failures=$((failures+1))
    fi
done

if [ $failures -ne 0 ]; then
    echo "Coverage check completed with $failures failures."
    exit 1
else
    echo "Coverage check completed successfully."
fi
