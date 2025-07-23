#!/bin/bash
# Build all tool modules

set -e

# Read modules from go.work
MODULES=$(go work edit -json | jq -r '.Use[].DiskPath' | sed 's|^\./||')

# Filter tool modules (exclude pkg/* and examples)
TOOL_MODULES=$(echo "$MODULES" | grep -v '^pkg/' | grep -v '^examples$')

for module in $TOOL_MODULES; do
    echo "Building $module..."
    if [ -d "$module/cmd" ]; then
        for cmd in "$module"/cmd/*; do
            if [ -d "$cmd" ]; then
                cmdname=$(basename "$cmd")
                echo "  Building $cmdname..."
                # Calculate relative path back to root for output
                outpath=$(echo "$module" | sed 's|[^/]*|..|g')/bin/$cmdname
                (cd "$module" && go build -o "$outpath" "./cmd/$cmdname")
            fi
        done
    fi
done
