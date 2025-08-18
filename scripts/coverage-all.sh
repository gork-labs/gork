#!/bin/bash
# Coverage check for all modules with failure collection and reporting

set -e

# Read modules from go.work
MODULES=$(go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

failures=0
failed_modules=""

for module in $MODULES; do
    echo "Checking coverage for $module..."
    # Use per-module thresholds defined in check-coverage.sh
    if ! ./scripts/check-coverage.sh "$module"; then
        failures=$((failures+1))
        if [ -z "$failed_modules" ]; then
            failed_modules="$module"
        else
            failed_modules="$failed_modules\n$module"
        fi
    fi
done

if [ $failures -ne 0 ]; then
    echo ""
    echo "❌ Coverage check completed with $failures failure(s)."
    echo "📋 Modules that failed coverage:"
    echo -e "$failed_modules" | sed 's/^/  - /'
    echo ""
    echo "💡 To fix coverage issues:"
    echo "   1. Review the coverage reports at <module>/coverage.html"
    echo "   2. Add tests for uncovered functions and edge cases"
    echo "   3. Run individual coverage check: ./scripts/check-coverage.sh <module>"
    exit 1
else
    echo "✅ Coverage check completed successfully for all modules." >&2
fi
