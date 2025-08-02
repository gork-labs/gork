#!/bin/bash
# Build all tool modules

set -e

TOOL_MODULES="cmd/*"

for cmd in $TOOL_MODULES; do
    echo "Building $cmd..."
    
    cmdname=$(basename "$cmd")
    
    go build -o "./bin/$cmdname" "./$cmd"
done
