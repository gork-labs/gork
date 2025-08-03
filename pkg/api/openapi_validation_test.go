package api

import (
	"reflect"
	"testing"
)

func TestParseValidationRule(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedKey string
		expectedVal string
	}{
		{
			name:        "rule with value",
			input:       "min=5",
			expectedKey: "min",
			expectedVal: "5",
		},
		{
			name:        "rule with complex value",
			input:       "regexp=^[a-zA-Z0-9]+$",
			expectedKey: "regexp",
			expectedVal: "^[a-zA-Z0-9]+$",
		},
		{
			name:        "rule without value",
			input:       "required",
			expectedKey: "required",
			expectedVal: "",
		},
		{
			name:        "rule with empty value",
			input:       "oneof=",
			expectedKey: "oneof",
			expectedVal: "",
		},
		{
			name:        "rule with multiple equals",
			input:       "regexp=a=b=c",
			expectedKey: "regexp",
			expectedVal: "a=b=c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val := parseValidationRule(tt.input)
			if key != tt.expectedKey {
				t.Errorf("parseValidationRule() key = %v, want %v", key, tt.expectedKey)
			}
			if val != tt.expectedVal {
				t.Errorf("parseValidationRule() val = %v, want %v", val, tt.expectedVal)
			}
		})
	}
}

func TestApplyValidationRule(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		val       string
		fieldType reflect.Type
		verify    func(*Schema) bool
		desc      string
	}{
		{
			name:      "min constraint on string",
			key:       "min",
			val:       "5",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MinLength != nil && *s.MinLength == 5
			},
			desc: "should set MinLength to 5",
		},
		{
			name:      "min constraint on int",
			key:       "min",
			val:       "10",
			fieldType: reflect.TypeOf(0),
			verify: func(s *Schema) bool {
				return s.Minimum != nil && *s.Minimum == 10.0
			},
			desc: "should set Minimum to 10",
		},
		{
			name:      "max constraint on string",
			key:       "max",
			val:       "100",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MaxLength != nil && *s.MaxLength == 100
			},
			desc: "should set MaxLength to 100",
		},
		{
			name:      "max constraint on float",
			key:       "max",
			val:       "99.5",
			fieldType: reflect.TypeOf(0.0),
			verify: func(s *Schema) bool {
				return s.Maximum != nil && *s.Maximum == 99.5
			},
			desc: "should set Maximum to 99.5",
		},
		{
			name:      "len constraint on string",
			key:       "len",
			val:       "8",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MinLength != nil && *s.MinLength == 8 &&
					s.MaxLength != nil && *s.MaxLength == 8
			},
			desc: "should set both MinLength and MaxLength to 8",
		},
		{
			name:      "regexp constraint",
			key:       "regexp",
			val:       "^[a-zA-Z0-9]+$",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.Pattern == "^[a-zA-Z0-9]+$"
			},
			desc: "should set Pattern",
		},
		{
			name:      "oneof constraint",
			key:       "oneof",
			val:       "red blue green",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return len(s.Enum) == 3 &&
					s.Enum[0] == "red" &&
					s.Enum[1] == "blue" &&
					s.Enum[2] == "green"
			},
			desc: "should set Enum values",
		},
		{
			name:      "gte constraint on int",
			key:       "gte",
			val:       "0",
			fieldType: reflect.TypeOf(0),
			verify: func(s *Schema) bool {
				return s.Minimum != nil && *s.Minimum == 0.0
			},
			desc: "should set Minimum to 0",
		},
		{
			name:      "lte constraint on float",
			key:       "lte",
			val:       "1000.5",
			fieldType: reflect.TypeOf(0.0),
			verify: func(s *Schema) bool {
				return s.Maximum != nil && *s.Maximum == 1000.5
			},
			desc: "should set Maximum to 1000.5",
		},
		{
			name:      "gt constraint on int",
			key:       "gt",
			val:       "5",
			fieldType: reflect.TypeOf(0),
			verify: func(s *Schema) bool {
				return s.Minimum != nil && *s.Minimum == 5.0
			},
			desc: "should set Minimum to 5",
		},
		{
			name:      "lt constraint on float",
			key:       "lt",
			val:       "10.5",
			fieldType: reflect.TypeOf(0.0),
			verify: func(s *Schema) bool {
				return s.Maximum != nil && *s.Maximum == 10.5
			},
			desc: "should set Maximum to 10.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{}
			applyValidationRule(schema, tt.key, tt.val, tt.fieldType)

			if !tt.verify(schema) {
				t.Errorf("applyValidationRule() failed: %s", tt.desc)
			}
		})
	}
}

func TestApplyMinConstraint(t *testing.T) {
	tests := []struct {
		name      string
		val       string
		fieldType reflect.Type
		verify    func(*Schema) bool
		desc      string
	}{
		{
			name:      "valid number for string type",
			val:       "5",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MinLength != nil && *s.MinLength == 5
			},
			desc: "should set MinLength for string type",
		},
		{
			name:      "valid number for int type",
			val:       "10",
			fieldType: reflect.TypeOf(0),
			verify: func(s *Schema) bool {
				return s.Minimum != nil && *s.Minimum == 10.0
			},
			desc: "should set Minimum for numeric type",
		},
		{
			name:      "valid float",
			val:       "3.14",
			fieldType: reflect.TypeOf(0.0),
			verify: func(s *Schema) bool {
				return s.Minimum != nil && *s.Minimum == 3.14
			},
			desc: "should set Minimum for float type",
		},
		{
			name:      "invalid number",
			val:       "not-a-number",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MinLength == nil && s.Minimum == nil
			},
			desc: "should not set any constraints for invalid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{}
			applyMinConstraint(schema, tt.val, tt.fieldType)

			if !tt.verify(schema) {
				t.Errorf("applyMinConstraint() failed: %s", tt.desc)
			}
		})
	}
}

func TestApplyMaxConstraint(t *testing.T) {
	tests := []struct {
		name      string
		val       string
		fieldType reflect.Type
		verify    func(*Schema) bool
		desc      string
	}{
		{
			name:      "valid number for string type",
			val:       "100",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MaxLength != nil && *s.MaxLength == 100
			},
			desc: "should set MaxLength for string type",
		},
		{
			name:      "valid number for int type",
			val:       "50",
			fieldType: reflect.TypeOf(0),
			verify: func(s *Schema) bool {
				return s.Maximum != nil && *s.Maximum == 50.0
			},
			desc: "should set Maximum for numeric type",
		},
		{
			name:      "valid float",
			val:       "99.99",
			fieldType: reflect.TypeOf(0.0),
			verify: func(s *Schema) bool {
				return s.Maximum != nil && *s.Maximum == 99.99
			},
			desc: "should set Maximum for float type",
		},
		{
			name:      "invalid number",
			val:       "invalid",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MaxLength == nil && s.Maximum == nil
			},
			desc: "should not set any constraints for invalid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{}
			applyMaxConstraint(schema, tt.val, tt.fieldType)

			if !tt.verify(schema) {
				t.Errorf("applyMaxConstraint() failed: %s", tt.desc)
			}
		})
	}
}

func TestApplyLenConstraint(t *testing.T) {
	tests := []struct {
		name      string
		val       string
		fieldType reflect.Type
		verify    func(*Schema) bool
		desc      string
	}{
		{
			name:      "valid length for string",
			val:       "8",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MinLength != nil && *s.MinLength == 8 &&
					s.MaxLength != nil && *s.MaxLength == 8
			},
			desc: "should set both MinLength and MaxLength",
		},
		{
			name:      "valid length for non-string type",
			val:       "5",
			fieldType: reflect.TypeOf([]string{}),
			verify: func(s *Schema) bool {
				// For non-string types, applyLenConstraint doesn't set anything
				return s.MinLength == nil && s.MaxLength == nil
			},
			desc: "should not set constraints for non-string types",
		},
		{
			name:      "invalid length",
			val:       "not-a-number",
			fieldType: reflect.TypeOf(""),
			verify: func(s *Schema) bool {
				return s.MinLength == nil && s.MaxLength == nil
			},
			desc: "should not set any constraints for invalid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{}
			applyLenConstraint(schema, tt.val, tt.fieldType)

			if !tt.verify(schema) {
				t.Errorf("applyLenConstraint() failed: %s", tt.desc)
			}
		})
	}
}

func TestApplyOneOfConstraint(t *testing.T) {
	tests := []struct {
		name   string
		val    string
		verify func(*Schema) bool
		desc   string
	}{
		{
			name: "space-separated values",
			val:  "red blue green",
			verify: func(s *Schema) bool {
				return len(s.Enum) == 3 &&
					s.Enum[0] == "red" &&
					s.Enum[1] == "blue" &&
					s.Enum[2] == "green"
			},
			desc: "should split on spaces and set Enum",
		},
		{
			name: "single value",
			val:  "single",
			verify: func(s *Schema) bool {
				return len(s.Enum) == 1 && s.Enum[0] == "single"
			},
			desc: "should handle single value",
		},
		{
			name: "empty value",
			val:  "",
			verify: func(s *Schema) bool {
				return len(s.Enum) == 0
			},
			desc: "should handle empty value",
		},
		{
			name: "values with extra spaces",
			val:  "  red   blue  green  ",
			verify: func(s *Schema) bool {
				return len(s.Enum) == 3 &&
					s.Enum[0] == "red" &&
					s.Enum[1] == "blue" &&
					s.Enum[2] == "green"
			},
			desc: "should trim spaces from values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{}
			applyOneOfConstraint(schema, tt.val)

			if !tt.verify(schema) {
				t.Errorf("applyOneOfConstraint() failed: %s", tt.desc)
			}
		})
	}
}

func TestIsStringKind(t *testing.T) {
	tests := []struct {
		name     string
		t        reflect.Type
		expected bool
	}{
		{
			name:     "string type",
			t:        reflect.TypeOf(""),
			expected: true,
		},
		{
			name:     "int type",
			t:        reflect.TypeOf(0),
			expected: false,
		},
		{
			name:     "float type",
			t:        reflect.TypeOf(0.0),
			expected: false,
		},
		{
			name:     "bool type",
			t:        reflect.TypeOf(true),
			expected: false,
		},
		{
			name:     "slice type",
			t:        reflect.TypeOf([]string{}),
			expected: false,
		},
		{
			name:     "struct type",
			t:        reflect.TypeOf(struct{}{}),
			expected: false,
		},
		{
			name:     "pointer to string",
			t:        reflect.TypeOf((*string)(nil)),
			expected: true,
		},
		{
			name:     "pointer to int",
			t:        reflect.TypeOf((*int)(nil)),
			expected: false,
		},
		{
			name:     "double pointer to string",
			t:        reflect.TypeOf((**string)(nil)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStringKind(tt.t)
			if result != tt.expected {
				t.Errorf("isStringKind() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestApplyValidationConstraints_Integration(t *testing.T) {
	// Test the integration with applyValidationConstraints function
	schema := &Schema{}
	parentSchema := &Schema{}
	field := reflect.StructField{Name: "TestField", Type: reflect.TypeOf("")}

	// Test with a complex validate tag
	validateTag := "required,min=5,max=100,regexp=^[a-zA-Z]+$"
	fieldType := reflect.TypeOf("")

	applyValidationConstraints(schema, validateTag, fieldType, parentSchema, field)

	// Should have applied all constraints
	if schema.MinLength == nil || *schema.MinLength != 5 {
		t.Error("Should have applied min constraint")
	}

	if schema.MaxLength == nil || *schema.MaxLength != 100 {
		t.Error("Should have applied max constraint")
	}

	if schema.Pattern != "^[a-zA-Z]+$" {
		t.Error("Should have applied regexp constraint")
	}
}

func TestValidationEdgeCases(t *testing.T) {
	t.Run("empty validation rule", func(t *testing.T) {
		schema := &Schema{}
		applyValidationRule(schema, "", "", reflect.TypeOf(""))

		// Should not panic or modify schema
		if schema.MinLength != nil || schema.MaxLength != nil || schema.Pattern != "" {
			t.Error("Empty rule should not modify schema")
		}
	})

	t.Run("unknown validation rule", func(t *testing.T) {
		schema := &Schema{}
		applyValidationRule(schema, "unknown", "value", reflect.TypeOf(""))

		// Should not panic or modify schema
		if schema.MinLength != nil || schema.MaxLength != nil || schema.Pattern != "" {
			t.Error("Unknown rule should not modify schema")
		}
	})

	t.Run("negative numbers", func(t *testing.T) {
		schema := &Schema{}
		applyMinConstraint(schema, "-5", reflect.TypeOf(""))

		if schema.MinLength == nil || *schema.MinLength != -5 {
			t.Error("Should handle negative numbers")
		}
	})

	t.Run("nil fieldSchema", func(t *testing.T) {
		// Test early return when fieldSchema is nil
		parentSchema := &Schema{}
		field := reflect.StructField{Name: "TestField", Type: reflect.TypeOf("")}

		// This should not panic and should return early
		applyValidationConstraints(nil, "required,min=5", reflect.TypeOf(""), parentSchema, field)

		// Parent schema should not be modified since fieldSchema was nil
		if len(parentSchema.Required) != 0 {
			t.Error("Parent schema should not be modified when fieldSchema is nil")
		}
	})
}

// TestWithRouteFilter tests the WithRouteFilter function
func TestWithRouteFilter(t *testing.T) {
	filterFunc := func(route *RouteInfo) bool {
		return route.Path != "/excluded"
	}

	option := WithRouteFilter(filterFunc)

	spec := &OpenAPISpec{}
	option(spec)

	// Test that the filter is applied correctly
	route1 := &RouteInfo{Path: "/included"}
	route2 := &RouteInfo{Path: "/excluded"}

	if !spec.routeFilter(route1) {
		t.Error("Expected /included route to pass filter")
	}

	if spec.routeFilter(route2) {
		t.Error("Expected /excluded route to be filtered out")
	}
}
