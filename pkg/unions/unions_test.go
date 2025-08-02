package unions

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/go-playground/validator/v10"
)

// Common test types to reduce duplication
type (
	// Basic types for simple tests
	StringData struct {
		Value string `json:"value" validate:"required"`
	}
	IntData struct {
		Value int `json:"value" validate:"required,min=1"`
	}
	BoolData struct {
		Value bool `json:"value" validate:"required"`
	}
	FloatData struct {
		Value float64 `json:"value" validate:"required"`
	}

	// Types with validation constraints
	EmailAuth struct {
		Type  string `json:"type" validate:"required,eq=email"`
		Email string `json:"email" validate:"required,email"`
	}
	PhoneAuth struct {
		Type  string `json:"type" validate:"required,eq=phone"`
		Phone string `json:"phone" validate:"required"`
	}
	TokenAuth struct {
		Type  string `json:"type" validate:"required,eq=token"`
		Token string `json:"token" validate:"required,uuid"`
	}
	UsernameAuth struct {
		Type     string `json:"type" validate:"required,eq=username"`
		Username string `json:"username" validate:"required,alphanum,min=3"`
	}

	// Complex nested types
	Address struct {
		Street string `json:"street" validate:"required"`
		City   string `json:"city" validate:"required"`
	}
	User struct {
		Name    string  `json:"name" validate:"required"`
		Email   string  `json:"email" validate:"required,email"`
		Address Address `json:"address" validate:"required"`
	}
	Company struct {
		Name    string  `json:"name" validate:"required"`
		TaxID   string  `json:"tax_id" validate:"required"`
		Address Address `json:"address" validate:"required"`
		Founded int     `json:"founded" validate:"required,min=1900"`
	}

	// Discriminator test types
	DiscriminatedStringType struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	DiscriminatedIntType struct {
		Type  string `json:"type"`
		Value int    `json:"value"`
	}
)

// Implement Discriminator interface for testing
func (d DiscriminatedStringType) DiscriminatorValue() string { return "string" }
func (d DiscriminatedIntType) DiscriminatorValue() string   { return "int" }

// Helper type that implements DiscriminatorField
type CustomDiscriminatorField struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

func (c CustomDiscriminatorField) DiscriminatorFieldName() string { return "kind" }
func (c CustomDiscriminatorField) DiscriminatorValue() string     { return "custom" }

// TestUnionUnmarshalJSON tests JSON unmarshaling for all union types
func TestUnionUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		expectedType string // "A", "B", "C", "D", or "error"
		wantErr      bool
	}{
		// Valid cases
		{"string value", `{"value": "hello"}`, "A", false},
		{"int value", `{"value": 1}`, "B", false},
		{"bool value", `{"value": true}`, "C", false},
		{"float value", `{"value": 3.14}`, "D", false},
		
		// Invalid cases
		{"invalid json", `{invalid}`, "error", true},
		{"empty object", `{}`, "error", true}, // fails validation due to required fields
		{"null", `null`, "error", true}, // null fails validation
		{"array", `[]`, "error", true},
		{"raw string", `"hello"`, "error", true},
		{"raw number", `42`, "error", true},
		{"boolean primitive", `true`, "error", true},
	}

	// Test Union2
	t.Run("Union2", func(t *testing.T) {
		for _, tt := range tests {
			if tt.expectedType == "C" || tt.expectedType == "D" { // Skip tests not applicable to Union2
				continue
			}
			t.Run(tt.name, func(t *testing.T) {
				var u Union2[StringData, IntData]
				err := json.Unmarshal([]byte(tt.json), &u)

				if (err != nil) != tt.wantErr {
					t.Errorf("Union2.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr {
					switch tt.expectedType {
					case "A":
						if u.A == nil {
							t.Error("expected A to be set")
						}
					case "B":
						if u.B == nil {
							t.Error("expected B to be set")
						}
					}
				}
			})
		}
	})

	// Test Union3
	t.Run("Union3", func(t *testing.T) {
		for _, tt := range tests {
			if tt.expectedType == "D" { // Skip tests not applicable to Union3
				continue
			}
			t.Run(tt.name, func(t *testing.T) {
				var u Union3[StringData, IntData, BoolData]
				err := json.Unmarshal([]byte(tt.json), &u)

				if (err != nil) != tt.wantErr {
					t.Errorf("Union3.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr {
					switch tt.expectedType {
					case "A":
						if u.A == nil {
							t.Error("expected A to be set")
						}
					case "B":
						if u.B == nil {
							t.Error("expected B to be set")
						}
					case "C":
						if u.C == nil {
							t.Error("expected C to be set")
						}
					}
				}
			})
		}
	})

	// Test Union4
	t.Run("Union4", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var u Union4[StringData, IntData, BoolData, FloatData]
				err := json.Unmarshal([]byte(tt.json), &u)

				if (err != nil) != tt.wantErr {
					t.Errorf("Union4.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr {
					switch tt.expectedType {
					case "A":
						if u.A == nil {
							t.Error("expected A to be set")
						}
					case "B":
						if u.B == nil {
							t.Error("expected B to be set")
						}
					case "C":
						if u.C == nil {
							t.Error("expected C to be set")
						}
					case "D":
						if u.D == nil {
							t.Error("expected D to be set")
						}
					}
				}
			})
		}
	})
}


// TestUnionValidation tests validation for all union types with complex validation scenarios
func TestUnionValidation(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		validOption string // which union option should succeed
	}{
		{"valid email", `{"type": "email", "email": "test@example.com"}`, false, "A"},
		{"valid phone", `{"type": "phone", "phone": "+1234567890"}`, false, "B"},
		{"valid token", `{"type": "token", "token": "550e8400-e29b-41d4-a716-446655440000"}`, false, "C"},
		{"valid username", `{"type": "username", "username": "user123"}`, false, "D"},
		
		// Invalid cases
		{"invalid email format", `{"type": "email", "email": "not-an-email"}`, true, ""},
		{"invalid token format", `{"type": "token", "token": "not-a-uuid"}`, true, ""},
		{"username too short", `{"type": "username", "username": "ab"}`, true, ""},
		{"type mismatch", `{"type": "email", "phone": "+1234567890"}`, true, ""},
		{"missing required field", `{"type": "email"}`, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u Union4[EmailAuth, PhoneAuth, TokenAuth, UsernameAuth]
			err := json.Unmarshal([]byte(tt.json), &u)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the correct option was set
				val, idx := u.Value()
				if val == nil {
					t.Error("Expected value to be set")
					return
				}

				expectedIdx := map[string]int{"A": 0, "B": 1, "C": 2, "D": 3}[tt.validOption]
				if idx != expectedIdx {
					t.Errorf("Expected option %s (index %d), got index %d", tt.validOption, expectedIdx, idx)
				}
			}
		})
	}
}

// TestUnionMarshalJSON tests JSON marshaling for all union types
func TestUnionMarshalJSON(t *testing.T) {
	t.Run("Union2", func(t *testing.T) {
		tests := []struct {
			name     string
			union    Union2[StringData, IntData]
			expected string
			wantErr  bool
		}{
			{"marshal A", Union2[StringData, IntData]{A: &StringData{Value: "hello"}}, `{"value":"hello"}`, false},
			{"marshal B", Union2[StringData, IntData]{B: &IntData{Value: 42}}, `{"value":42}`, false},
			{"empty union", Union2[StringData, IntData]{}, "", true},
			{"both set (A priority)", Union2[StringData, IntData]{A: &StringData{Value: "hello"}, B: &IntData{Value: 42}}, `{"value":"hello"}`, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := json.Marshal(tt.union)
				if (err != nil) != tt.wantErr {
					t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && string(data) != tt.expected {
					t.Errorf("MarshalJSON() = %q, want %q", string(data), tt.expected)
				}
			})
		}
	})

	t.Run("Union3", func(t *testing.T) {
		tests := []struct {
			name     string
			union    Union3[StringData, IntData, BoolData]
			expected string
			wantErr  bool
		}{
			{"marshal A", Union3[StringData, IntData, BoolData]{A: &StringData{Value: "test"}}, `{"value":"test"}`, false},
			{"marshal B", Union3[StringData, IntData, BoolData]{B: &IntData{Value: 123}}, `{"value":123}`, false},
			{"marshal C", Union3[StringData, IntData, BoolData]{C: &BoolData{Value: true}}, `{"value":true}`, false},
			{"empty union", Union3[StringData, IntData, BoolData]{}, "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := json.Marshal(tt.union)
				if (err != nil) != tt.wantErr {
					t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && string(data) != tt.expected {
					t.Errorf("MarshalJSON() = %q, want %q", string(data), tt.expected)
				}
			})
		}
	})

	t.Run("Union4", func(t *testing.T) {
		tests := []struct {
			name     string
			union    Union4[StringData, IntData, BoolData, FloatData]
			expected string
			wantErr  bool
		}{
			{"marshal A", Union4[StringData, IntData, BoolData, FloatData]{A: &StringData{Value: "value"}}, `{"value":"value"}`, false},
			{"marshal B", Union4[StringData, IntData, BoolData, FloatData]{B: &IntData{Value: 42}}, `{"value":42}`, false},
			{"marshal C", Union4[StringData, IntData, BoolData, FloatData]{C: &BoolData{Value: true}}, `{"value":true}`, false},
			{"marshal D", Union4[StringData, IntData, BoolData, FloatData]{D: &FloatData{Value: 3.14}}, `{"value":3.14}`, false},
			{"empty union", Union4[StringData, IntData, BoolData, FloatData]{}, "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := json.Marshal(tt.union)
				if (err != nil) != tt.wantErr {
					t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && string(data) != tt.expected {
					t.Errorf("MarshalJSON() = %q, want %q", string(data), tt.expected)
				}
			})
		}
	})
}



// TestUnionValue tests the Value method for all union types
func TestUnionValue(t *testing.T) {
	t.Run("Union2", func(t *testing.T) {
		tests := []struct {
			name      string
			union     Union2[StringData, IntData]
			wantIndex int
			wantNil   bool
		}{
			{"A is set", Union2[StringData, IntData]{A: &StringData{Value: "test"}}, 0, false},
			{"B is set", Union2[StringData, IntData]{B: &IntData{Value: 42}}, 1, false},
			{"nothing set", Union2[StringData, IntData]{}, -1, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				value, index := tt.union.Value()
				if index != tt.wantIndex {
					t.Errorf("Value() index = %v, want %v", index, tt.wantIndex)
				}
				if tt.wantNil && value != nil {
					t.Errorf("Value() = %v, want nil", value)
				} else if !tt.wantNil && value == nil {
					t.Error("Value() = nil, want non-nil")
				}
			})
		}
	})

	t.Run("Union3", func(t *testing.T) {
		tests := []struct {
			name      string
			union     Union3[StringData, IntData, BoolData]
			wantIndex int
		}{
			{"A is set", Union3[StringData, IntData, BoolData]{A: &StringData{Value: "test"}}, 0},
			{"B is set", Union3[StringData, IntData, BoolData]{B: &IntData{Value: 42}}, 1},
			{"C is set", Union3[StringData, IntData, BoolData]{C: &BoolData{Value: true}}, 2},
			{"nothing set", Union3[StringData, IntData, BoolData]{}, -1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, index := tt.union.Value()
				if index != tt.wantIndex {
					t.Errorf("Value() index = %v, want %v", index, tt.wantIndex)
				}
			})
		}
	})

	t.Run("Union4", func(t *testing.T) {
		tests := []struct {
			name      string
			union     Union4[StringData, IntData, BoolData, FloatData]
			wantIndex int
		}{
			{"A is set", Union4[StringData, IntData, BoolData, FloatData]{A: &StringData{Value: "test"}}, 0},
			{"B is set", Union4[StringData, IntData, BoolData, FloatData]{B: &IntData{Value: 42}}, 1},
			{"C is set", Union4[StringData, IntData, BoolData, FloatData]{C: &BoolData{Value: true}}, 2},
			{"D is set", Union4[StringData, IntData, BoolData, FloatData]{D: &FloatData{Value: 3.14}}, 3},
			{"nothing set", Union4[StringData, IntData, BoolData, FloatData]{}, -1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, index := tt.union.Value()
				if index != tt.wantIndex {
					t.Errorf("Value() index = %v, want %v", index, tt.wantIndex)
				}
			})
		}
	})
}



// TestUnionValidateMethod tests the Validate method for all union types
func TestUnionValidateMethod(t *testing.T) {
	validate := validator.New()

	t.Run("Union2", func(t *testing.T) {
		tests := []struct {
			name    string
			union   Union2[EmailAuth, PhoneAuth]
			wantErr bool
			errMsg  string
		}{
			{"no value set", Union2[EmailAuth, PhoneAuth]{}, true, "exactly one union option must be set"},
			{"both values set", Union2[EmailAuth, PhoneAuth]{
				A: &EmailAuth{Type: "email", Email: "test@example.com"},
				B: &PhoneAuth{Type: "phone", Phone: "+1234567890"},
			}, true, "only one union option can be set"},
			{"valid A", Union2[EmailAuth, PhoneAuth]{A: &EmailAuth{Type: "email", Email: "test@example.com"}}, false, ""},
			{"invalid A", Union2[EmailAuth, PhoneAuth]{A: &EmailAuth{Type: "email", Email: "invalid-email"}}, true, ""},
			{"valid B", Union2[EmailAuth, PhoneAuth]{B: &PhoneAuth{Type: "phone", Phone: "+1234567890"}}, false, ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.union.Validate(validate)
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			})
		}
	})

	t.Run("Union3", func(t *testing.T) {
		tests := []struct {
			name    string
			union   Union3[StringData, IntData, BoolData]
			wantErr bool
		}{
			{"no value set", Union3[StringData, IntData, BoolData]{}, true},
			{"multiple values set", Union3[StringData, IntData, BoolData]{
				A: &StringData{Value: "test"},
				B: &IntData{Value: 20},
			}, true},
			{"valid A", Union3[StringData, IntData, BoolData]{A: &StringData{Value: "test"}}, false},
			{"invalid B (min validation)", Union3[StringData, IntData, BoolData]{B: &IntData{Value: 0}}, true}, // min=1 required
			{"valid C", Union3[StringData, IntData, BoolData]{C: &BoolData{Value: true}}, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.union.Validate(validate)
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("Union4", func(t *testing.T) {
		tests := []struct {
			name    string
			union   Union4[StringData, IntData, BoolData, FloatData]
			wantErr bool
			errMsg  string
		}{
			{"no value set", Union4[StringData, IntData, BoolData, FloatData]{}, true, "exactly one union option must be set"},
			{"all values set", Union4[StringData, IntData, BoolData, FloatData]{
				A: &StringData{Value: "test"},
				B: &IntData{Value: 1},
				C: &BoolData{Value: true},
				D: &FloatData{Value: 3.14},
			}, true, "only one union option can be set"},
			{"valid D", Union4[StringData, IntData, BoolData, FloatData]{D: &FloatData{Value: 3.14}}, false, ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.union.Validate(validate)
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			})
		}
	})
}



// TestUnionComplexTypes tests union types with complex nested structures
func TestUnionComplexTypes(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectedIdx int // 0 = User, 1 = Company, -1 = error
		wantErr     bool
	}{
		{
			name: "valid user",
			json: `{
				"name": "John Doe",
				"email": "john@example.com",
				"address": {
					"street": "123 Main St",
					"city": "New York"
				}
			}`,
			expectedIdx: 0,
			wantErr:     false,
		},
		{
			name: "valid company",
			json: `{
				"name": "Acme Corp",
				"tax_id": "12-3456789",
				"address": {
					"street": "456 Business Ave",
					"city": "San Francisco"
				},
				"founded": 2020
			}`,
			expectedIdx: 1,
			wantErr:     false,
		},
		{
			name: "invalid user (missing email)",
			json: `{
				"name": "John Doe",
				"address": {
					"street": "123 Main St",
					"city": "New York"
				}
			}`,
			expectedIdx: -1,
			wantErr:     true,
		},
		{
			name: "invalid company (founded too early)",
			json: `{
				"name": "Old Corp",
				"tax_id": "12-3456789",
				"address": {
					"street": "456 Business Ave",
					"city": "San Francisco"
				},
				"founded": 1800
			}`,
			expectedIdx: -1,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u Union2[User, Company]
			err := json.Unmarshal([]byte(tt.json), &u)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				_, idx := u.Value()
				if idx != tt.expectedIdx {
					t.Errorf("Expected index %d, got %d", tt.expectedIdx, idx)
				}
			}
		})
	}
}

// TestValidatorConcurrency tests concurrent access to validator singleton
func TestValidatorConcurrency(t *testing.T) {
	// Reset the validator to ensure we're testing initialization
	validatorInstance = nil
	validatorOnce = sync.Once{}

	const goroutines = 100
	validators := make([]*validator.Validate, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			validators[idx] = getValidator()
		}(i)
	}

	wg.Wait()

	// Check that all goroutines got the same instance
	first := validators[0]
	for i := 1; i < goroutines; i++ {
		if validators[i] != first {
			t.Errorf("Got different validator instance at index %d", i)
		}
	}
}

// TestUnionEdgeCases tests various edge cases for union types
func TestUnionEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		desc    string
	}{
		{"null value", `null`, true, "null should fail validation"},
		{"empty object", `{}`, true, "empty object should fail validation"},
		{"array", `[]`, true, "array should fail unmarshaling"},
		{"boolean primitive", `true`, true, "boolean primitive should fail unmarshaling"},
		{"raw string", `"hello"`, true, "raw string should fail unmarshaling"},
		{"raw number", `42`, true, "raw number should fail unmarshaling"},
		{"number as string", `{"value": "123"}`, false, "flexible type conversion should work"},
		{"string as number", `{"value": 123}`, false, "flexible type conversion should work"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with simple types that allow flexible conversion
			type FlexibleString struct {
				Value any `json:"value" validate:"required"`
			}
			type FlexibleNumber struct {
				Value any `json:"value" validate:"required"`
			}

			var u Union2[FlexibleString, FlexibleNumber]
			err := json.Unmarshal([]byte(tt.json), &u)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v (%s)", err, tt.wantErr, tt.desc)
			}
		})
	}
}

// TestDiscriminatorInterfaces tests the discriminator interfaces
func TestDiscriminatorInterfaces(t *testing.T) {
	// Test basic discriminator interface
	t.Run("Discriminator interface", func(t *testing.T) {
		strType := DiscriminatedStringType{Type: "string", Value: "test"}
		intType := DiscriminatedIntType{Type: "int", Value: 42}

		if strType.DiscriminatorValue() != "string" {
			t.Errorf("Expected discriminator value 'string', got %s", strType.DiscriminatorValue())
		}

		if intType.DiscriminatorValue() != "int" {
			t.Errorf("Expected discriminator value 'int', got %s", intType.DiscriminatorValue())
		}
	})

	// Test custom discriminator field
	t.Run("DiscriminatorField interface", func(t *testing.T) {
		customType := CustomDiscriminatorField{Kind: "custom", Value: "test"}

		if customType.DiscriminatorFieldName() != "kind" {
			t.Errorf("Expected discriminator field name 'kind', got %s", customType.DiscriminatorFieldName())
		}

		if customType.DiscriminatorValue() != "custom" {
			t.Errorf("Expected discriminator value 'custom', got %s", customType.DiscriminatorValue())
		}
	})

	// Test discriminator-like behavior in unions (using existing auth types)
	t.Run("Discriminator behavior in unions", func(t *testing.T) {
		tests := []struct {
			name     string
			json     string
			wantType string
			wantErr  bool
		}{
			{"email auth", `{"type": "email", "email": "user@example.com"}`, "email", false},
			{"phone auth", `{"type": "phone", "phone": "+1234567890"}`, "phone", false},
			{"token auth", `{"type": "token", "token": "550e8400-e29b-41d4-a716-446655440000"}`, "token", false},
			{"mismatched type field", `{"type": "email", "phone": "+1234567890"}`, "", true},
			{"invalid token format", `{"type": "token", "token": "not-a-uuid"}`, "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var u Union3[EmailAuth, PhoneAuth, TokenAuth]
				err := json.Unmarshal([]byte(tt.json), &u)

				if (err != nil) != tt.wantErr {
					t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr {
					val, idx := u.Value()
					if val == nil {
						t.Error("Expected value to be set")
						return
					}

					// Verify correct type was selected based on discriminator
					switch idx {
					case 0:
						if u.A.Type != tt.wantType {
							t.Errorf("Expected type %s, got %s", tt.wantType, u.A.Type)
						}
					case 1:
						if u.B.Type != tt.wantType {
							t.Errorf("Expected type %s, got %s", tt.wantType, u.B.Type)
						}
					case 2:
						if u.C.Type != tt.wantType {
							t.Errorf("Expected type %s, got %s", tt.wantType, u.C.Type)
						}
					}
				}
			})
		}
	})
}

// TestUnionComplexPayloads tests union types with complex payload validation
func TestUnionComplexPayloads(t *testing.T) {
	type StringPayload struct {
		Type  string `json:"type" validate:"required,eq=string"`
		Value string `json:"value" validate:"required"`
	}
	type NumberPayload struct {
		Type  string  `json:"type" validate:"required,eq=number"`
		Value float64 `json:"value" validate:"required"`
	}
	type BoolPayload struct {
		Type  string `json:"type" validate:"required,eq=bool"`
		Value bool   `json:"value"`
	}
	type ArrayPayload struct {
		Type  string   `json:"type" validate:"required,eq=array"`
		Value []string `json:"value" validate:"required,dive,required"`
	}

	tests := []struct {
		name        string
		json        string
		expectedIdx int // 0=String, 1=Number, 2=Bool, 3=Array, -1=error
		wantErr     bool
		validation  func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload])
	}{
		{"string payload", `{"type": "string", "value": "hello world"}`, 0, false, func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
			if u.A == nil || u.A.Value != "hello world" {
				t.Error("Expected string payload with value 'hello world'")
			}
		}},
		{"number payload", `{"type": "number", "value": 42.5}`, 1, false, func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
			if u.B == nil || u.B.Value != 42.5 {
				t.Error("Expected number payload with value 42.5")
			}
		}},
		{"bool payload", `{"type": "bool", "value": true}`, 2, false, func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
			if u.C == nil || u.C.Value != true {
				t.Error("Expected bool payload with value true")
			}
		}},
		{"array payload", `{"type": "array", "value": ["a", "b", "c"]}`, 3, false, func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
			if u.D == nil || len(u.D.Value) != 3 {
				t.Error("Expected array payload with 3 elements")
			}
		}},
		
		// Error cases
		{"type mismatch", `{"type": "string", "value": 123}`, -1, true, nil},
		{"missing required field", `{"type": "number"}`, -1, true, nil},
		{"invalid type value", `{"type": "object", "value": {}}`, -1, true, nil},
		{"empty array element", `{"type": "array", "value": [""]}`, -1, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]
			err := json.Unmarshal([]byte(tt.json), &u)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				_, idx := u.Value()
				if idx != tt.expectedIdx {
					t.Errorf("Expected index %d, got %d", tt.expectedIdx, idx)
				}
				if tt.validation != nil {
					tt.validation(t, u)
				}
			}
		})
	}
}

// TestUnionConcurrency tests concurrent operations on union types
func TestUnionConcurrency(t *testing.T) {
	const goroutines = 100

	// Test concurrent marshaling
	t.Run("concurrent marshal", func(t *testing.T) {
		u := Union2[StringData, IntData]{A: &StringData{Value: "test"}}

		var wg sync.WaitGroup
		errors := make(chan error, goroutines)

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := json.Marshal(u)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent marshal error: %v", err)
		}
	})

	// Test concurrent unmarshaling
	t.Run("concurrent unmarshal", func(t *testing.T) {
		jsonData := `{"value": "test"}`

		var wg sync.WaitGroup
		errors := make(chan error, goroutines)

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var u Union2[StringData, IntData]
				err := json.Unmarshal([]byte(jsonData), &u)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent unmarshal error: %v", err)
		}
	})

	// Test concurrent Value() calls
	t.Run("concurrent value calls", func(t *testing.T) {
		u := Union3[StringData, IntData, BoolData]{B: &IntData{Value: 42}}

		var wg sync.WaitGroup
		results := make(chan int, goroutines)

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, idx := u.Value()
				results <- idx
			}()
		}

		wg.Wait()
		close(results)

		// All results should be index 1 (B is set)
		for idx := range results {
			if idx != 1 {
				t.Errorf("Expected index 1, got %d", idx)
			}
		}
	})
}
