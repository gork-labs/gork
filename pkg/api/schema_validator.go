package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gork-labs/gork/pkg/gorkson"
)

// OpenAPISchema is a type alias for gorkson.OpenAPISchema for backward compatibility with schema validation.
type OpenAPISchema = gorkson.OpenAPISchema

// Constants from gorkson for backward compatibility.
const (
	OpenAPITypeString  = gorkson.OpenAPITypeString
	OpenAPITypeInteger = gorkson.OpenAPITypeInteger
	OpenAPITypeNumber  = gorkson.OpenAPITypeNumber
	OpenAPITypeBoolean = gorkson.OpenAPITypeBoolean
)

// Error function aliases for backward compatibility.
var (
	NewInvalidFormatError = gorkson.NewParseErrorWithReason
)

// validatePattern validates value against regex pattern constraint.
func validatePattern(value string, pattern string, typeName string) error {
	if pattern == "" {
		return nil
	}

	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return NewInvalidFormatError(typeName, value, fmt.Sprintf("invalid regex pattern in schema: %v", err))
	}
	if !matched {
		return NewInvalidFormatError(typeName, value, fmt.Sprintf("value does not match required pattern: %s", pattern))
	}

	return nil
}

// validateStringLength validates string length constraints.
func validateStringLength(value string, minLength, maxLength *int, typeName string) error {
	valueLen := len(value)

	if minLength != nil && valueLen < *minLength {
		return NewInvalidFormatError(typeName, value, fmt.Sprintf("string too short: minimum length is %d, got %d", *minLength, valueLen))
	}

	if maxLength != nil && valueLen > *maxLength {
		return NewInvalidFormatError(typeName, value, fmt.Sprintf("string too long: maximum length is %d, got %d", *maxLength, valueLen))
	}

	return nil
}

// validateNumeric validates numeric value format and range constraints.
func validateNumeric(value string, schemaType string, minimum, maximum *float64, typeName string) error {
	var numValue float64
	var err error

	if schemaType == OpenAPITypeInteger {
		intVal, parseErr := strconv.ParseInt(value, 10, 64)
		if parseErr != nil {
			return NewInvalidFormatError(typeName, value, fmt.Sprintf("invalid integer format: %v", parseErr))
		}
		numValue = float64(intVal)
	} else {
		numValue, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return NewInvalidFormatError(typeName, value, fmt.Sprintf("invalid number format: %v", err))
		}
	}

	// Validate range constraints
	if minimum != nil && numValue < *minimum {
		return NewInvalidFormatError(typeName, value, fmt.Sprintf("number too small: minimum is %g, got %g", *minimum, numValue))
	}

	if maximum != nil && numValue > *maximum {
		return NewInvalidFormatError(typeName, value, fmt.Sprintf("number too large: maximum is %g, got %g", *maximum, numValue))
	}

	return nil
}

// validateEnum validates value against enum constraints.
func validateEnum(value string, enumValues []interface{}, typeName string) error {
	if len(enumValues) == 0 {
		return nil
	}

	for _, enumValue := range enumValues {
		if enumStr, ok := enumValue.(string); ok && enumStr == value {
			return nil // Found in enum
		}
		// Also check non-string enum values converted to string
		if fmt.Sprintf("%v", enumValue) == value {
			return nil // Found in enum
		}
	}

	// Not found in enum
	enumStrs := make([]string, len(enumValues))
	for i, enumValue := range enumValues {
		if str, ok := enumValue.(string); ok {
			enumStrs[i] = str
		} else {
			enumStrs[i] = fmt.Sprintf("%v", enumValue)
		}
	}
	return NewInvalidFormatError(typeName, value, fmt.Sprintf("value not in allowed enum: %s, allowed values: [%s]", value, strings.Join(enumStrs, ", ")))
}

// validateValueAgainstSchema performs automatic validation based on OpenAPI schema constraints.
// It validates pattern, length, range, and enum constraints before codec parsing.
func validateValueAgainstSchema(value string, schema OpenAPISchema, typeName string) error {
	// Pattern validation
	if err := validatePattern(value, schema.Pattern, typeName); err != nil {
		return err
	}

	// String length validation
	if schema.Type == OpenAPITypeString {
		if err := validateStringLength(value, schema.MinLength, schema.MaxLength, typeName); err != nil {
			return err
		}
	}

	// Numeric validation for integer and number types
	if schema.Type == OpenAPITypeInteger || schema.Type == OpenAPITypeNumber {
		if err := validateNumeric(value, schema.Type, schema.Minimum, schema.Maximum, typeName); err != nil {
			return err
		}
	}

	// Enum validation
	if err := validateEnum(value, schema.Enum, typeName); err != nil {
		return err
	}

	return nil
}
