#!/bin/bash

# Script to validate OpenAPI schemas using Swagger validator API
# Usage: ./scripts/validate-openapi.sh <file>

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <openapi-spec-file>"
    echo "Example: $0 examples/openapi.yaml"
    exit 1
fi

SPEC_FILE="$1"

if [ ! -f "$SPEC_FILE" ]; then
    echo "Error: File '$SPEC_FILE' not found"
    exit 1
fi

echo "Validating OpenAPI spec: $SPEC_FILE"

# Determine content type based on file extension
if [[ "$SPEC_FILE" == *.yaml ]] || [[ "$SPEC_FILE" == *.yml ]]; then
    CONTENT_TYPE="application/yaml"
else
    CONTENT_TYPE="application/json"
fi

# Send the file to Swagger validator
response=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: $CONTENT_TYPE" \
    --data-binary "@$SPEC_FILE" \
    "https://validator.swagger.io/validator/debug")

# Extract HTTP status code (last line)
http_code=$(echo "$response" | tail -n1)

# Extract response body (all lines except last)
body=$(echo "$response" | sed '$d')

# Check if validation succeeded
if [ "$http_code" != "200" ]; then
    echo "Error: Validation failed with HTTP status $http_code"
    echo "Response: $body"
    exit 1
fi

# Parse the JSON response to check for errors
if command -v jq &> /dev/null; then
    # If jq is available, use it for pretty printing
    echo "$body" | jq '.'
    
    # Check if response is empty object (valid spec)
    if [ "$body" = "{}" ]; then
        # Empty response means no errors
        echo "✓ No validation errors found"
    else
        # Check if there are any errors in the messages array
        error_count=$(echo "$body" | jq '.messages // [] | map(select(.level == "error")) | length')
        if [ "$error_count" -gt 0 ]; then
            echo "Error: Found $error_count validation errors"
            exit 1
        fi
    fi
else
    # If jq is not available, just print the raw response
    echo "$body"
    
    # Check if response is empty object (valid spec)
    if [ "$body" = "{}" ]; then
        echo "✓ No validation errors found"
    elif echo "$body" | grep -q '"level":"error"'; then
        # Do a simple grep check for error level
        echo "Error: Validation errors found"
        exit 1
    fi
fi

echo "✓ OpenAPI spec is valid!"