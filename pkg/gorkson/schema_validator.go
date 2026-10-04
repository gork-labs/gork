package gorkson

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// validatePattern validates value against regex pattern constraint.
func validatePattern(value string, pattern string, typeName string) error {
	if pattern == "" {
		return nil
	}

	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return NewParseErrorWithReason(typeName, value, fmt.Sprintf("invalid regex pattern in schema: %v", err))
	}
	if !matched {
		return NewParseErrorWithReason(typeName, value, fmt.Sprintf("value does not match required pattern: %s", pattern))
	}

	return nil
}

// validateStringLength validates string length constraints.
func validateStringLength(value string, minLength, maxLength *int, typeName string) error {
	valueLen := len(value)

	if minLength != nil && valueLen < *minLength {
		return NewParseErrorWithReason(typeName, value, fmt.Sprintf("string too short: minimum length is %d, got %d", *minLength, valueLen))
	}

	if maxLength != nil && valueLen > *maxLength {
		return NewParseErrorWithReason(typeName, value, fmt.Sprintf("string too long: maximum length is %d, got %d", *maxLength, valueLen))
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
			return NewParseErrorWithReason(typeName, value, fmt.Sprintf("invalid integer format: %v", parseErr))
		}
		numValue = float64(intVal)
	} else {
		numValue, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return NewParseErrorWithReason(typeName, value, fmt.Sprintf("invalid number format: %v", err))
		}
	}

	// Validate range constraints
	if minimum != nil && numValue < *minimum {
		return NewParseErrorWithReason(typeName, value, fmt.Sprintf("number too small: minimum is %g, got %g", *minimum, numValue))
	}

	if maximum != nil && numValue > *maximum {
		return NewParseErrorWithReason(typeName, value, fmt.Sprintf("number too large: maximum is %g, got %g", *maximum, numValue))
	}

	return nil
}

// validateEnum validates value against enum constraints.
func validateEnum(value string, enumValues []interface{}, typeName string) error {
	if len(enumValues) == 0 {
		return nil
	}

	allowed := make([]string, len(enumValues))
	for i, enumValue := range enumValues {
		allowed[i] = fmt.Sprint(enumValue)
		if allowed[i] == value {
			return nil
		}
	}

	return NewParseErrorWithReason(typeName, value, fmt.Sprintf("value not in allowed enum: %s, allowed values: [%s]", value, strings.Join(allowed, ", ")))
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
