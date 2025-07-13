package generator

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidatorMapper maps go-playground/validator tags to OpenAPI constraints
type ValidatorMapper struct {
	customValidators map[string]string // Maps custom validator names to descriptions
}

// NewValidatorMapper creates a new validator mapper
func NewValidatorMapper() *ValidatorMapper {
	return &ValidatorMapper{
		customValidators: make(map[string]string),
	}
}

// RegisterCustomValidator registers a custom validator with a description
func (vm *ValidatorMapper) RegisterCustomValidator(name, description string) {
	vm.customValidators[name] = description
}

// MapValidatorTags maps validator tags to OpenAPI schema properties
func (vm *ValidatorMapper) MapValidatorTags(validateTag string, schema *Schema, fieldType string) error {
	if validateTag == "" {
		return nil
	}

	// Split by comma but handle nested validations
	tags := vm.splitValidatorTags(validateTag)
	
	var orConditions []string
	
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		
		// Handle OR conditions
		if strings.Contains(tag, "|") {
			orConditions = strings.Split(tag, "|")
			// Process OR conditions separately
			vm.processOrConditions(orConditions, schema, fieldType)
			continue
		}
		
		// Parse tag and value
		tagName, tagValue := vm.parseTag(tag)
		
		switch tagName {
		case "required":
			// This is handled in IsRequired function
		case "omitempty":
			// Field is optional
		case "email":
			schema.Format = "email"
		case "url", "uri":
			schema.Format = "uri"
		case "uuid", "uuid3", "uuid4", "uuid5":
			schema.Format = "uuid"
		case "datetime":
			schema.Format = "date-time"
		case "date":
			schema.Format = "date"
		case "time":
			schema.Format = "time"
		case "duration":
			schema.Format = "duration"
		case "ipv4", "ip":
			schema.Format = "ipv4"
		case "ipv6":
			schema.Format = "ipv6"
		case "cidr":
			schema.Format = "cidr"
		case "mac":
			schema.Pattern = `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
		case "hostname":
			schema.Format = "hostname"
		case "fqdn":
			schema.Format = "hostname"
			schema.Pattern = `^([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*\.)+[a-zA-Z]{2,}$`
		case "base64":
			schema.Format = "byte"
		case "base64url":
			schema.Format = "byte"
			if schema.Description == "" {
				schema.Description = "Base64 URL-safe encoded"
			}
		case "e164":
			schema.Pattern = `^\+[1-9]\d{1,14}$`
			if schema.Description == "" {
				schema.Description = "E.164 phone number format"
			}
		case "alpha":
			schema.Pattern = `^[a-zA-Z]+$`
		case "alphanum":
			schema.Pattern = `^[a-zA-Z0-9]+$`
		case "numeric":
			schema.Pattern = `^[0-9]+$`
		case "hexadecimal":
			schema.Pattern = `^[0-9a-fA-F]+$`
		case "hexcolor":
			schema.Pattern = `^#[0-9a-fA-F]{6}$`
		case "rgb":
			schema.Pattern = `^rgb\((\d{1,3},\s*){2}\d{1,3}\)$`
		case "rgba":
			schema.Pattern = `^rgba\((\d{1,3},\s*){3}(0|1|0?\.\d+)\)$`
		case "latitude":
			if schema.Type == "number" || schema.Type == "integer" {
				min := -90.0
				max := 90.0
				schema.Minimum = &min
				schema.Maximum = &max
			}
		case "longitude":
			if schema.Type == "number" || schema.Type == "integer" {
				min := -180.0
				max := 180.0
				schema.Minimum = &min
				schema.Maximum = &max
			}
		case "iso3166_1_alpha2":
			schema.Pattern = `^[A-Z]{2}$`
			if schema.Description == "" {
				schema.Description = "ISO 3166-1 alpha-2 country code"
			}
		case "iso3166_1_alpha3":
			schema.Pattern = `^[A-Z]{3}$`
			if schema.Description == "" {
				schema.Description = "ISO 3166-1 alpha-3 country code"
			}
		
		// Numeric constraints
		case "min", "gte":
			if val, err := strconv.ParseFloat(tagValue, 64); err == nil {
				if fieldType == "string" {
					intVal := int(val)
					schema.MinLength = &intVal
				} else if schema.Type == "array" {
					intVal := int(val)
					schema.MinItems = &intVal
				} else {
					schema.Minimum = &val
				}
			}
		case "max", "lte":
			if val, err := strconv.ParseFloat(tagValue, 64); err == nil {
				if fieldType == "string" {
					intVal := int(val)
					schema.MaxLength = &intVal
				} else if schema.Type == "array" {
					intVal := int(val)
					schema.MaxItems = &intVal
				} else {
					schema.Maximum = &val
				}
			}
		case "gt":
			if val, err := strconv.ParseFloat(tagValue, 64); err == nil {
				schema.Minimum = &val
				schema.ExclusiveMinimum = true
			}
		case "lt":
			if val, err := strconv.ParseFloat(tagValue, 64); err == nil {
				schema.Maximum = &val
				schema.ExclusiveMaximum = true
			}
		case "len":
			if val, err := strconv.Atoi(tagValue); err == nil {
				if fieldType == "string" {
					schema.MinLength = &val
					schema.MaxLength = &val
				} else if schema.Type == "array" {
					schema.MinItems = &val
					schema.MaxItems = &val
				}
			}
		case "oneof":
			// Parse enum values
			if tagValue != "" {
				values := strings.Fields(tagValue)
				schema.Enum = make([]interface{}, len(values))
				for i, v := range values {
					schema.Enum[i] = v
				}
			}
		case "contains":
			if tagValue != "" {
				schema.Pattern = fmt.Sprintf(".*%s.*", escapeRegex(tagValue))
			}
		case "excludes":
			if tagValue != "" {
				schema.Pattern = fmt.Sprintf("^((?!%s).)*$", escapeRegex(tagValue))
			}
		case "startswith":
			if tagValue != "" {
				schema.Pattern = fmt.Sprintf("^%s", escapeRegex(tagValue))
			}
		case "endswith":
			if tagValue != "" {
				schema.Pattern = fmt.Sprintf("%s$", escapeRegex(tagValue))
			}
		case "unique":
			schema.UniqueItems = true
		
		// Cross-field validation (add to description)
		case "eqfield", "nefield", "gtfield", "gtefield", "ltfield", "ltefield",
		     "eqcsfield", "necsfield", "gtcsfield", "gtecsfield", "ltcsfield", "ltecsfield":
			desc := fmt.Sprintf("Must be %s field '%s'", strings.TrimSuffix(tagName, "field"), tagValue)
			if schema.Description != "" {
				schema.Description += ". " + desc
			} else {
				schema.Description = desc
			}
		
		// Conditional validation
		case "required_if", "required_unless", "required_with", "required_with_all",
		     "required_without", "required_without_all", "excluded_with", "excluded_without":
			desc := fmt.Sprintf("Conditional validation: %s %s", tagName, tagValue)
			if schema.Description != "" {
				schema.Description += ". " + desc
			} else {
				schema.Description = desc
			}
		
		// dive - for nested validation
		case "dive":
			// Mark that array/map items should have validation applied
			if schema.Type == "array" && schema.Items != nil {
				// Validation will be applied to items
			}
		
		// Custom validators
		default:
			if desc, ok := vm.customValidators[tagName]; ok {
				if schema.Description != "" {
					schema.Description += ". " + desc
				} else {
					schema.Description = desc
				}
			}
		}
	}
	
	return nil
}

// processOrConditions handles OR validation conditions using anyOf
func (vm *ValidatorMapper) processOrConditions(conditions []string, schema *Schema, fieldType string) {
	if len(conditions) <= 1 {
		return
	}
	
	anyOfSchemas := make([]*Schema, 0, len(conditions))
	
	for _, condition := range conditions {
		subSchema := &Schema{
			Type: schema.Type,
		}
		vm.MapValidatorTags(strings.TrimSpace(condition), subSchema, fieldType)
		anyOfSchemas = append(anyOfSchemas, subSchema)
	}
	
	schema.AnyOf = anyOfSchemas
}

// splitValidatorTags splits validation tags while respecting nested structures
func (vm *ValidatorMapper) splitValidatorTags(tag string) []string {
	if tag == "" {
		return []string{}
	}
	
	var tags []string
	var current strings.Builder
	depth := 0
	
	for _, ch := range tag {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				tags = append(tags, current.String())
				current.Reset()
				continue
			}
		}
		current.WriteRune(ch)
	}
	
	if current.Len() > 0 {
		tags = append(tags, current.String())
	}
	
	return tags
}

// parseTag parses a validation tag into name and value
func (vm *ValidatorMapper) parseTag(tag string) (name, value string) {
	parts := strings.SplitN(tag, "=", 2)
	name = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}
	return
}

// escapeRegex escapes special regex characters
func escapeRegex(s string) string {
	// Handle backslash first to avoid double escaping
	result := strings.ReplaceAll(s, "\\", "\\\\")
	
	// Then escape other special regex characters
	special := []string{".", "+", "*", "?", "^", "$", "(", ")", "[", "]", "{", "}", "|"}
	for _, char := range special {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// IsRequired checks if a field is required based on validate tags
func IsRequired(validateTag string) bool {
	if validateTag == "" {
		return false
	}
	
	tags := strings.Split(validateTag, ",")
	hasRequired := false
	hasOmitempty := false
	
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "required" {
			hasRequired = true
		} else if tag == "omitempty" {
			hasOmitempty = true
		}
	}
	
	return hasRequired && !hasOmitempty
}