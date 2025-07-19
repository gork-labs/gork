package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorTagMapping(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		fieldType string
		expected  func(*Schema)
	}{
		// Basic validators
		{
			name:      "email format",
			tag:       "email",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Format = "email"
			},
		},
		{
			name:      "uuid format",
			tag:       "uuid",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Format = "uuid"
			},
		},
		{
			name:      "datetime format",
			tag:       "datetime",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Format = "date-time"
			},
		},
		{
			name:      "uri format",
			tag:       "uri",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Format = "uri"
			},
		},
		{
			name:      "ipv4 format",
			tag:       "ipv4",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Format = "ipv4"
			},
		},
		{
			name:      "ipv6 format",
			tag:       "ipv6",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Format = "ipv6"
			},
		},
		
		// String constraints
		{
			name:      "string min length",
			tag:       "min=3",
			fieldType: "string",
			expected: func(s *Schema) {
				minLen := 3
				s.MinLength = &minLen
			},
		},
		{
			name:      "string max length",
			tag:       "max=50",
			fieldType: "string",
			expected: func(s *Schema) {
				maxLen := 50
				s.MaxLength = &maxLen
			},
		},
		{
			name:      "string exact length",
			tag:       "len=10",
			fieldType: "string",
			expected: func(s *Schema) {
				len := 10
				s.MinLength = &len
				s.MaxLength = &len
			},
		},
		
		// Numeric constraints
		{
			name:      "numeric minimum",
			tag:       "gte=0",
			fieldType: "number",
			expected: func(s *Schema) {
				min := 0.0
				s.Minimum = &min
			},
		},
		{
			name:      "numeric maximum",
			tag:       "lte=100",
			fieldType: "number",
			expected: func(s *Schema) {
				max := 100.0
				s.Maximum = &max
			},
		},
		{
			name:      "exclusive minimum",
			tag:       "gt=0",
			fieldType: "number",
			expected: func(s *Schema) {
				min := 0.0
				s.Minimum = &min
				s.ExclusiveMinimum = true
			},
		},
		{
			name:      "exclusive maximum",
			tag:       "lt=100",
			fieldType: "number",
			expected: func(s *Schema) {
				max := 100.0
				s.Maximum = &max
				s.ExclusiveMaximum = true
			},
		},
		
		// Array constraints
		{
			name:      "array min items",
			tag:       "min=1",
			fieldType: "array",
			expected: func(s *Schema) {
				s.Type = "array"
				minItems := 1
				s.MinItems = &minItems
			},
		},
		{
			name:      "array max items",
			tag:       "max=10",
			fieldType: "array",
			expected: func(s *Schema) {
				s.Type = "array"
				maxItems := 10
				s.MaxItems = &maxItems
			},
		},
		
		// Enum values
		{
			name:      "enum values",
			tag:       "oneof=admin user moderator",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Enum = []interface{}{"admin", "user", "moderator"}
			},
		},
		
		// Pattern validators
		{
			name:      "alpha pattern",
			tag:       "alpha",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^[a-zA-Z]+$`
			},
		},
		{
			name:      "alphanum pattern",
			tag:       "alphanum",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^[a-zA-Z0-9]+$`
			},
		},
		{
			name:      "numeric pattern",
			tag:       "numeric",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^[0-9]+$`
			},
		},
		{
			name:      "hexadecimal pattern",
			tag:       "hexadecimal",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^[0-9a-fA-F]+$`
			},
		},
		{
			name:      "hexcolor pattern",
			tag:       "hexcolor",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^#[0-9a-fA-F]{6}$`
			},
		},
		{
			name:      "mac pattern",
			tag:       "mac",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
			},
		},
		{
			name:      "e164 phone pattern",
			tag:       "e164",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^\+[1-9]\d{1,14}$`
				s.Description = "E.164 phone number format"
			},
		},
		
		// Geographic coordinates
		{
			name:      "latitude bounds",
			tag:       "latitude",
			fieldType: "number",
			expected: func(s *Schema) {
				s.Type = "number"
				min := -90.0
				max := 90.0
				s.Minimum = &min
				s.Maximum = &max
			},
		},
		{
			name:      "longitude bounds",
			tag:       "longitude",
			fieldType: "number",
			expected: func(s *Schema) {
				s.Type = "number"
				min := -180.0
				max := 180.0
				s.Minimum = &min
				s.Maximum = &max
			},
		},
		
		// ISO codes
		{
			name:      "iso3166 alpha2",
			tag:       "iso3166_1_alpha2",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^[A-Z]{2}$`
				s.Description = "ISO 3166-1 alpha-2 country code"
			},
		},
		
		// String content validators
		{
			name:      "contains",
			tag:       "contains=hello",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `.*hello.*`
			},
		},
		{
			name:      "excludes",
			tag:       "excludes=bad",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^((?!bad).)*$`
			},
		},
		{
			name:      "startswith",
			tag:       "startswith=prefix",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `^prefix`
			},
		},
		{
			name:      "endswith",
			tag:       "endswith=suffix",
			fieldType: "string",
			expected: func(s *Schema) {
				s.Pattern = `suffix$`
			},
		},
		
		// Array unique items
		{
			name:      "unique items",
			tag:       "unique",
			fieldType: "array",
			expected: func(s *Schema) {
				s.UniqueItems = true
			},
		},
	}

	mapper := NewValidatorMapper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: tt.fieldType}
			switch tt.fieldType {
			case "array":
				schema.Type = "array"
			case "number":
				schema.Type = "number"
			}
			
			err := mapper.MapValidatorTags(tt.tag, schema, tt.fieldType)
			require.NoError(t, err)
			
			expected := &Schema{Type: schema.Type}
			tt.expected(expected)
			
			assert.Equal(t, expected, schema)
		})
	}
}

func TestComplexValidatorTags(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		fieldType string
		check     func(*testing.T, *Schema)
	}{
		{
			name:      "required email with length",
			tag:       "required,email,min=5,max=255",
			fieldType: "string",
			check: func(t *testing.T, s *Schema) {
				assert.Equal(t, "email", s.Format)
				assert.Equal(t, 5, *s.MinLength)
				assert.Equal(t, 255, *s.MaxLength)
			},
		},
		{
			name:      "numeric range with pattern",
			tag:       "gte=0,lte=100,numeric",
			fieldType: "string",
			check: func(t *testing.T, s *Schema) {
				assert.Equal(t, 0, *s.MinLength) // min for string type
				assert.Equal(t, 100, *s.MaxLength) // max for string type
				assert.Equal(t, `^[0-9]+$`, s.Pattern)
			},
		},
		{
			name:      "array with dive validation",
			tag:       "min=1,max=10,dive,alphanum",
			fieldType: "array",
			check: func(t *testing.T, s *Schema) {
				assert.Equal(t, "array", s.Type)
				assert.Equal(t, 1, *s.MinItems)
				assert.Equal(t, 10, *s.MaxItems)
			},
		},
	}

	mapper := NewValidatorMapper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: tt.fieldType}
			if tt.fieldType == "array" {
				schema.Type = "array"
			}
			
			err := mapper.MapValidatorTags(tt.tag, schema, tt.fieldType)
			require.NoError(t, err)
			
			tt.check(t, schema)
		})
	}
}

func TestCrossFieldValidation(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		description string
	}{
		{
			name:        "equal field",
			tag:         "eqfield=Password",
			description: "Must be eq field 'Password'",
		},
		{
			name:        "not equal field",
			tag:         "nefield=Username",
			description: "Must be ne field 'Username'",
		},
		{
			name:        "greater than field",
			tag:         "gtfield=MinValue",
			description: "Must be gt field 'MinValue'",
		},
		{
			name:        "less than field",
			tag:         "ltfield=MaxValue",
			description: "Must be lt field 'MaxValue'",
		},
	}

	mapper := NewValidatorMapper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: "string"}
			err := mapper.MapValidatorTags(tt.tag, schema, "string")
			require.NoError(t, err)
			assert.Contains(t, schema.Description, tt.description)
		})
	}
}

func TestConditionalValidation(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{
			name: "required if",
			tag:  "required_if=Type admin",
		},
		{
			name: "required unless",
			tag:  "required_unless=Status active",
		},
		{
			name: "required with",
			tag:  "required_with=Email",
		},
		{
			name: "required without",
			tag:  "required_without=Phone",
		},
		{
			name: "excluded with",
			tag:  "excluded_with=Phone",
		},
	}

	mapper := NewValidatorMapper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: "string"}
			err := mapper.MapValidatorTags(tt.tag, schema, "string")
			require.NoError(t, err)
			assert.Contains(t, schema.Description, "Conditional validation")
		})
	}
}

func TestCustomValidatorDetection(t *testing.T) {
	mapper := NewValidatorMapper()
	
	// Register custom validators
	mapper.RegisterCustomValidator("username", "Must be a valid username")
	mapper.RegisterCustomValidator("strongpassword", "Must meet strong password requirements")
	
	tests := []struct {
		name     string
		tag      string
		hasDesc  bool
		expected string
	}{
		{
			name:     "known custom validator",
			tag:      "username",
			hasDesc:  true,
			expected: "Must be a valid username",
		},
		{
			name:     "unknown custom validator",
			tag:      "customthing",
			hasDesc:  false,
		},
		{
			name:     "mixed validators with custom",
			tag:      "required,username,min=3",
			hasDesc:  true,
			expected: "Must be a valid username",
		},
		{
			name:     "multiple custom validators",
			tag:      "username,strongpassword",
			hasDesc:  true,
			expected: "Must be a valid username. Must meet strong password requirements",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: "string"}
			err := mapper.MapValidatorTags(tt.tag, schema, "string")
			require.NoError(t, err)
			
			if tt.hasDesc {
				assert.NotEmpty(t, schema.Description)
				if tt.expected != "" {
					assert.Contains(t, schema.Description, tt.expected)
				}
			} else {
				assert.Empty(t, schema.Description)
			}
		})
	}
}

func TestOrConditions(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		fieldType string
		check     func(*testing.T, *Schema)
	}{
		{
			name:      "rgb or rgba",
			tag:       "rgb|rgba",
			fieldType: "string",
			check: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.AnyOf)
				assert.Len(t, s.AnyOf, 2)
				
				// Check that both patterns are set
				rgbPattern := `^rgb\((\d{1,3},\s*){2}\d{1,3}\)$`
				rgbaPattern := `^rgba\((\d{1,3},\s*){3}(0|1|0?\.\d+)\)$`
				
				patterns := []string{s.AnyOf[0].Pattern, s.AnyOf[1].Pattern}
				assert.Contains(t, patterns, rgbPattern)
				assert.Contains(t, patterns, rgbaPattern)
			},
		},
		{
			name:      "email or phone",
			tag:       "email|e164",
			fieldType: "string",
			check: func(t *testing.T, s *Schema) {
				require.NotNil(t, s.AnyOf)
				assert.Len(t, s.AnyOf, 2)
				
				// One should have email format, other should have e164 pattern
				hasEmail := false
				hasE164 := false
				for _, schema := range s.AnyOf {
					if schema.Format == "email" {
						hasEmail = true
					}
					if schema.Pattern == `^\+[1-9]\d{1,14}$` {
						hasE164 = true
					}
				}
				assert.True(t, hasEmail)
				assert.True(t, hasE164)
			},
		},
	}

	mapper := NewValidatorMapper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: tt.fieldType}
			err := mapper.MapValidatorTags(tt.tag, schema, tt.fieldType)
			require.NoError(t, err)
			
			tt.check(t, schema)
		})
	}
}

func TestIsRequired(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "required field",
			tag:      "required",
			expected: true,
		},
		{
			name:     "required with other validators",
			tag:      "required,email,max=255",
			expected: true,
		},
		{
			name:     "omitempty field",
			tag:      "omitempty,email",
			expected: false,
		},
		{
			name:     "required but omitempty overrides",
			tag:      "required,omitempty,email",
			expected: false,
		},
		{
			name:     "no validation tags",
			tag:      "",
			expected: false,
		},
		{
			name:     "only other validators",
			tag:      "email,max=255",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRequired(tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitValidatorTags(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected []string
	}{
		{
			name:     "simple tags",
			tag:      "required,email,max=255",
			expected: []string{"required", "email", "max=255"},
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: []string{},
		},
		{
			name:     "single tag",
			tag:      "required",
			expected: []string{"required"},
		},
		{
			name:     "tags with spaces",
			tag:      "required, email , max=255",
			expected: []string{"required", " email ", " max=255"},
		},
		{
			name:     "tags with parentheses",
			tag:      "oneof=red blue,contains=(test),min=1",
			expected: []string{"oneof=red blue", "contains=(test)", "min=1"},
		},
	}

	mapper := NewValidatorMapper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.splitValidatorTags(tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		wantName  string
		wantValue string
	}{
		{
			name:      "tag without value",
			tag:       "required",
			wantName:  "required",
			wantValue: "",
		},
		{
			name:      "tag with value",
			tag:       "max=255",
			wantName:  "max",
			wantValue: "255",
		},
		{
			name:      "tag with complex value",
			tag:       "oneof=admin user moderator",
			wantName:  "oneof",
			wantValue: "admin user moderator",
		},
		{
			name:      "tag with equals in value",
			tag:       "contains==test",
			wantName:  "contains",
			wantValue: "=test",
		},
		{
			name:      "tag with spaces",
			tag:       " max = 255 ",
			wantName:  "max",
			wantValue: "255",
		},
	}

	mapper := NewValidatorMapper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, value := mapper.parseTag(tt.tag)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestEscapeRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "dot",
			input:    "hello.world",
			expected: "hello\\.world",
		},
		{
			name:     "multiple special chars",
			input:    "a.b+c*d?e^f$g",
			expected: "a\\.b\\+c\\*d\\?e\\^f\\$g",
		},
		{
			name:     "brackets and parens",
			input:    "test(1)[2]{3}",
			expected: "test\\(1\\)\\[2\\]\\{3\\}",
		},
		{
			name:     "pipe and backslash",
			input:    "a|b\\c",
			expected: "a\\|b\\\\c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeRegex(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}