package unions

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/go-playground/validator/v10"
)

// Test Union2 basic functionality
func TestUnion2(t *testing.T) {
	type StringType struct {
		Value string `json:"value"`
	}
	
	type IntType struct {
		Value int `json:"value"`
	}
	
	tests := []struct {
		name    string
		json    string
		wantA   bool
		wantB   bool
		wantErr bool
	}{
		{
			name:  "unmarshal string type",
			json:  `{"value": "hello"}`,
			wantA: true,
		},
		{
			name:  "unmarshal int type",
			json:  `{"value": 42}`,
			wantB: true,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u Union2[StringType, IntType]
			err := json.Unmarshal([]byte(tt.json), &u)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantA && u.A == nil {
				t.Error("expected A to be set")
			}
			if tt.wantB && u.B == nil {
				t.Error("expected B to be set")
			}
		})
	}
}

// Test Union3 basic functionality
func TestUnion3(t *testing.T) {
	type StringType struct {
		Value string `json:"value"`
	}
	
	type IntType struct {
		Value int `json:"value"`
	}
	
	type BoolType struct {
		Value bool `json:"value"`
	}
	
	// Test marshaling
	u := Union3[StringType, IntType, BoolType]{
		B: &IntType{Value: 42},
	}
	
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	
	expected := `{"value":42}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
	
	// Test unmarshaling
	var u2 Union3[StringType, IntType, BoolType]
	err = json.Unmarshal(data, &u2)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	
	if u2.B == nil || u2.B.Value != 42 {
		t.Error("Unmarshaling failed")
	}
}

// Test Union4 validation
func TestUnion4Validation(t *testing.T) {
	type Option1 struct {
		Email string `json:"email" validate:"required,email"`
	}
	
	type Option2 struct {
		Phone string `json:"phone" validate:"required,e164"`
	}
	
	type Option3 struct {
		Username string `json:"username" validate:"required,alphanum,min=3"`
	}
	
	type Option4 struct {
		ID int `json:"id" validate:"required,min=1"`
	}
	
	// Test invalid email - should try other options
	invalidEmail := `{"email": "not-an-email"}`
	var u Union4[Option1, Option2, Option3, Option4]
	err := json.Unmarshal([]byte(invalidEmail), &u)
	
	// Should fail because it doesn't match any valid option
	if err == nil {
		t.Error("Expected validation error for invalid email")
	}
	
	// Test valid username
	validUsername := `{"username": "user123"}`
	err = json.Unmarshal([]byte(validUsername), &u)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if u.C == nil || u.C.Username != "user123" {
		t.Error("Expected option C to be set")
	}
}

// Test Union2 MarshalJSON
func TestUnion2_MarshalJSON(t *testing.T) {
	type StringType struct {
		Value string `json:"value"`
	}
	
	type IntType struct {
		Value int `json:"value"`
	}
	
	tests := []struct {
		name     string
		union    Union2[StringType, IntType]
		expected string
		wantErr  bool
	}{
		{
			name:     "marshal A value",
			union:    Union2[StringType, IntType]{A: &StringType{Value: "hello"}},
			expected: `{"value":"hello"}`,
		},
		{
			name:     "marshal B value",
			union:    Union2[StringType, IntType]{B: &IntType{Value: 42}},
			expected: `{"value":42}`,
		},
		{
			name:    "marshal empty union",
			union:   Union2[StringType, IntType]{},
			wantErr: true,
		},
		{
			name:     "marshal with both values set (should marshal A)",
			union:    Union2[StringType, IntType]{A: &StringType{Value: "hello"}, B: &IntType{Value: 42}},
			expected: `{"value":"hello"}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.union)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(data), tt.expected)
			}
		})
	}
}

// Test Union3 MarshalJSON
func TestUnion3_MarshalJSON(t *testing.T) {
	type A struct {
		A string `json:"a"`
	}
	type B struct {
		B int `json:"b"`
	}
	type C struct {
		C bool `json:"c"`
	}
	
	tests := []struct {
		name     string
		union    Union3[A, B, C]
		expected string
		wantErr  bool
	}{
		{
			name:     "marshal A",
			union:    Union3[A, B, C]{A: &A{A: "test"}},
			expected: `{"a":"test"}`,
		},
		{
			name:     "marshal B",
			union:    Union3[A, B, C]{B: &B{B: 123}},
			expected: `{"b":123}`,
		},
		{
			name:     "marshal C",
			union:    Union3[A, B, C]{C: &C{C: true}},
			expected: `{"c":true}`,
		},
		{
			name:    "empty union",
			union:   Union3[A, B, C]{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.union)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(data), tt.expected)
			}
		})
	}
}

// Test Union4 MarshalJSON
func TestUnion4_MarshalJSON(t *testing.T) {
	type A struct{ Field string `json:"field"` }
	type B struct{ Field int `json:"field"` }
	type C struct{ Field bool `json:"field"` }
	type D struct{ Field float64 `json:"field"` }
	
	tests := []struct {
		name     string
		union    Union4[A, B, C, D]
		expected string
		wantErr  bool
	}{
		{
			name:     "marshal A",
			union:    Union4[A, B, C, D]{A: &A{Field: "value"}},
			expected: `{"field":"value"}`,
		},
		{
			name:     "marshal B",
			union:    Union4[A, B, C, D]{B: &B{Field: 42}},
			expected: `{"field":42}`,
		},
		{
			name:     "marshal C",
			union:    Union4[A, B, C, D]{C: &C{Field: true}},
			expected: `{"field":true}`,
		},
		{
			name:     "marshal D",
			union:    Union4[A, B, C, D]{D: &D{Field: 3.14}},
			expected: `{"field":3.14}`,
		},
		{
			name:    "empty union",
			union:   Union4[A, B, C, D]{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.union)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(data), tt.expected)
			}
		})
	}
}

// Test Union2 Value method
func TestUnion2_Value(t *testing.T) {
	type A struct{ Value string }
	type B struct{ Value int }
	
	tests := []struct {
		name      string
		union     Union2[A, B]
		wantValue interface{}
		wantIndex int
	}{
		{
			name:      "A is set",
			union:     Union2[A, B]{A: &A{Value: "test"}},
			wantValue: &A{Value: "test"},
			wantIndex: 0,
		},
		{
			name:      "B is set",
			union:     Union2[A, B]{B: &B{Value: 42}},
			wantValue: &B{Value: 42},
			wantIndex: 1,
		},
		{
			name:      "nothing set",
			union:     Union2[A, B]{},
			wantValue: nil,
			wantIndex: -1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, index := tt.union.Value()
			if index != tt.wantIndex {
				t.Errorf("Value() index = %v, want %v", index, tt.wantIndex)
			}
			if tt.wantValue == nil && value != nil {
				t.Errorf("Value() = %v, want nil", value)
			}
		})
	}
}

// Test Union3 Value method
func TestUnion3_Value(t *testing.T) {
	type A struct{ Val string }
	type B struct{ Val int }
	type C struct{ Val bool }
	
	tests := []struct {
		name      string
		union     Union3[A, B, C]
		wantIndex int
	}{
		{
			name:      "A is set",
			union:     Union3[A, B, C]{A: &A{Val: "test"}},
			wantIndex: 0,
		},
		{
			name:      "B is set",
			union:     Union3[A, B, C]{B: &B{Val: 42}},
			wantIndex: 1,
		},
		{
			name:      "C is set",
			union:     Union3[A, B, C]{C: &C{Val: true}},
			wantIndex: 2,
		},
		{
			name:      "nothing set",
			union:     Union3[A, B, C]{},
			wantIndex: -1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, index := tt.union.Value()
			if index != tt.wantIndex {
				t.Errorf("Value() index = %v, want %v", index, tt.wantIndex)
			}
		})
	}
}

// Test Union4 Value method
func TestUnion4_Value(t *testing.T) {
	type A struct{}
	type B struct{}
	type C struct{}
	type D struct{}
	
	tests := []struct {
		name      string
		union     Union4[A, B, C, D]
		wantIndex int
	}{
		{
			name:      "A is set",
			union:     Union4[A, B, C, D]{A: &A{}},
			wantIndex: 0,
		},
		{
			name:      "B is set",
			union:     Union4[A, B, C, D]{B: &B{}},
			wantIndex: 1,
		},
		{
			name:      "C is set",
			union:     Union4[A, B, C, D]{C: &C{}},
			wantIndex: 2,
		},
		{
			name:      "D is set",
			union:     Union4[A, B, C, D]{D: &D{}},
			wantIndex: 3,
		},
		{
			name:      "nothing set",
			union:     Union4[A, B, C, D]{},
			wantIndex: -1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, index := tt.union.Value()
			if index != tt.wantIndex {
				t.Errorf("Value() index = %v, want %v", index, tt.wantIndex)
			}
		})
	}
}

// Test Union2 Validate method
func TestUnion2_Validate(t *testing.T) {
	type A struct {
		Email string `validate:"required,email"`
	}
	type B struct {
		Phone string `validate:"required,e164"`
	}
	
	validate := validator.New()
	
	tests := []struct {
		name    string
		union   Union2[A, B]
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no value set",
			union:   Union2[A, B]{},
			wantErr: true,
			errMsg:  "exactly one union option must be set",
		},
		{
			name:    "both values set",
			union:   Union2[A, B]{A: &A{Email: "test@example.com"}, B: &B{Phone: "+1234567890"}},
			wantErr: true,
			errMsg:  "only one union option can be set",
		},
		{
			name:  "valid A",
			union: Union2[A, B]{A: &A{Email: "test@example.com"}},
		},
		{
			name:    "invalid A",
			union:   Union2[A, B]{A: &A{Email: "invalid-email"}},
			wantErr: true,
		},
		{
			name:  "valid B",
			union: Union2[A, B]{B: &B{Phone: "+1234567890"}},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.union.Validate(validate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// Test Union3 Validate method
func TestUnion3_Validate(t *testing.T) {
	type A struct {
		Val string `validate:"required"`
	}
	type B struct {
		Val int `validate:"required,min=10"`
	}
	type C struct {
		Val bool
	}
	
	validate := validator.New()
	
	tests := []struct {
		name    string
		union   Union3[A, B, C]
		wantErr bool
	}{
		{
			name:    "no value set",
			union:   Union3[A, B, C]{},
			wantErr: true,
		},
		{
			name:    "multiple values set",
			union:   Union3[A, B, C]{A: &A{Val: "test"}, B: &B{Val: 20}},
			wantErr: true,
		},
		{
			name:  "valid A",
			union: Union3[A, B, C]{A: &A{Val: "test"}},
		},
		{
			name:    "invalid B (min validation)",
			union:   Union3[A, B, C]{B: &B{Val: 5}},
			wantErr: true,
		},
		{
			name:  "valid C",
			union: Union3[A, B, C]{C: &C{Val: true}},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.union.Validate(validate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test Union4 Validate method
func TestUnion4_Validate(t *testing.T) {
	type A struct{ Val string }
	type B struct{ Val int }
	type C struct{ Val bool }
	type D struct{ Val float64 }
	
	validate := validator.New()
	
	tests := []struct {
		name    string
		union   Union4[A, B, C, D]
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no value set",
			union:   Union4[A, B, C, D]{},
			wantErr: true,
			errMsg:  "exactly one union option must be set",
		},
		{
			name:    "all values set",
			union:   Union4[A, B, C, D]{A: &A{}, B: &B{}, C: &C{}, D: &D{}},
			wantErr: true,
			errMsg:  "only one union option can be set",
		},
		{
			name:  "valid D",
			union: Union4[A, B, C, D]{D: &D{Val: 3.14}},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.union.Validate(validate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// Test complex nested types
func TestUnion2_ComplexTypes(t *testing.T) {
	type Address struct {
		Street string `json:"street" validate:"required"`
		City   string `json:"city" validate:"required"`
	}
	
	type User struct {
		Name    string  `json:"name" validate:"required"`
		Email   string  `json:"email" validate:"required,email"`
		Address Address `json:"address" validate:"required"`
	}
	
	type Company struct {
		Name    string    `json:"name" validate:"required"`
		TaxID   string    `json:"tax_id" validate:"required"`
		Address Address   `json:"address" validate:"required"`
		Founded int       `json:"founded" validate:"required,min=1900"`
	}
	
	// Test valid user
	userJSON := `{
		"name": "John Doe",
		"email": "john@example.com",
		"address": {
			"street": "123 Main St",
			"city": "New York"
		}
	}`
	
	var u1 Union2[User, Company]
	err := json.Unmarshal([]byte(userJSON), &u1)
	if err != nil {
		t.Errorf("Failed to unmarshal valid user: %v", err)
	}
	if u1.A == nil {
		t.Error("Expected User to be set")
	}
	
	// Test invalid user (missing email)
	invalidUserJSON := `{
		"name": "John Doe",
		"address": {
			"street": "123 Main St",
			"city": "New York"
		}
	}`
	
	var u2 Union2[User, Company]
	err = json.Unmarshal([]byte(invalidUserJSON), &u2)
	if err == nil {
		t.Error("Expected error for invalid user")
	}
	
	// Test valid company
	companyJSON := `{
		"name": "Acme Corp",
		"tax_id": "12-3456789",
		"address": {
			"street": "456 Business Ave",
			"city": "San Francisco"
		},
		"founded": 2020
	}`
	
	var u3 Union2[User, Company]
	err = json.Unmarshal([]byte(companyJSON), &u3)
	if err != nil {
		t.Errorf("Failed to unmarshal valid company: %v", err)
	}
	if u3.B == nil {
		t.Error("Expected Company to be set")
	}
}

// Test concurrent access to validator singleton
func TestGetValidator_Concurrent(t *testing.T) {
	// Reset the validator to ensure we're testing initialization
	validatorInstance = nil
	validatorOnce = sync.Once{}
	
	const goroutines = 100
	validators := make([]*validator.Validate, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	
	for i := 0; i < goroutines; i++ {
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

// Test UnmarshalJSON with various JSON types
func TestUnion2_UnmarshalJSON_EdgeCases(t *testing.T) {
	type NumType struct {
		Value float64 `json:"value"`
	}
	
	type StrType struct {
		Value string `json:"value"`
	}
	
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "null value",
			json: `null`,
			wantErr: false, // null can be unmarshaled into the struct but all fields will be nil
		},
		{
			name: "empty object",
			json: `{}`,
			wantErr: false, // Empty object can be unmarshaled but will fail validation
		},
		{
			name: "array",
			json: `[]`,
			wantErr: true,
		},
		{
			name: "number as string",
			json: `{"value": "123"}`,
			wantErr: false,
		},
		{
			name: "string as number",
			json: `{"value": 123}`,
			wantErr: false,
		},
		{
			name: "boolean",
			json: `true`,
			wantErr: true,
		},
		{
			name: "raw string",
			json: `"hello"`,
			wantErr: true,
		},
		{
			name: "raw number",
			json: `42`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u Union2[NumType, StrType]
			err := json.Unmarshal([]byte(tt.json), &u)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test discriminator-like behavior
func TestUnion3_DiscriminatorBehavior(t *testing.T) {
	type EmailAuth struct {
		Type  string `json:"type" validate:"required,eq=email"`
		Email string `json:"email" validate:"required,email"`
	}
	
	type PhoneAuth struct {
		Type  string `json:"type" validate:"required,eq=phone"`
		Phone string `json:"phone" validate:"required"`
	}
	
	type TokenAuth struct {
		Type  string `json:"type" validate:"required,eq=token"`
		Token string `json:"token" validate:"required,uuid"`
	}
	
	tests := []struct {
		name     string
		json     string
		wantType string
		wantErr  bool
	}{
		{
			name:     "email auth",
			json:     `{"type": "email", "email": "user@example.com"}`,
			wantType: "email",
		},
		{
			name:     "phone auth",
			json:     `{"type": "phone", "phone": "+1234567890"}`,
			wantType: "phone",
		},
		{
			name:     "token auth",
			json:     `{"type": "token", "token": "550e8400-e29b-41d4-a716-446655440000"}`,
			wantType: "token",
		},
		{
			name:    "mismatched type field",
			json:    `{"type": "email", "phone": "+1234567890"}`,
			wantErr: true,
		},
		{
			name:    "invalid token format",
			json:    `{"type": "token", "token": "not-a-uuid"}`,
			wantErr: true,
		},
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
}


// Test Union4 with all edge cases
func TestUnion4_CompleteScenarios(t *testing.T) {
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
		name      string
		json      string
		checkFunc func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload])
		wantErr   bool
	}{
		{
			name: "string payload",
			json: `{"type": "string", "value": "hello world"}`,
			checkFunc: func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
				if u.A == nil || u.A.Value != "hello world" {
					t.Error("Expected string payload with value 'hello world'")
				}
			},
		},
		{
			name: "number payload",
			json: `{"type": "number", "value": 42.5}`,
			checkFunc: func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
				if u.B == nil || u.B.Value != 42.5 {
					t.Error("Expected number payload with value 42.5")
				}
			},
		},
		{
			name: "bool payload",
			json: `{"type": "bool", "value": true}`,
			checkFunc: func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
				if u.C == nil || u.C.Value != true {
					t.Error("Expected bool payload with value true")
				}
			},
		},
		{
			name: "array payload",
			json: `{"type": "array", "value": ["a", "b", "c"]}`,
			checkFunc: func(t *testing.T, u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]) {
				if u.D == nil || len(u.D.Value) != 3 {
					t.Error("Expected array payload with 3 elements")
				}
			},
		},
		{
			name:    "type mismatch",
			json:    `{"type": "string", "value": 123}`,
			wantErr: true,
		},
		{
			name:    "missing required field",
			json:    `{"type": "number"}`,
			wantErr: true,
		},
		{
			name:    "invalid type value",
			json:    `{"type": "object", "value": {}}`,
			wantErr: true,
		},
		{
			name:    "empty array (validation should fail)",
			json:    `{"type": "array", "value": [""]}`, // Need at least one element with dive,required validation
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u Union4[StringPayload, NumberPayload, BoolPayload, ArrayPayload]
			err := json.Unmarshal([]byte(tt.json), &u)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, u)
			}
		})
	}
}

// Test for potential race conditions in union operations
func TestUnion2_ConcurrentOperations(t *testing.T) {
	type A struct {
		Value string `json:"value"`
	}
	
	type B struct {
		Value int `json:"value"`
	}
	
	// Test concurrent marshaling
	t.Run("concurrent marshal", func(t *testing.T) {
		u := Union2[A, B]{A: &A{Value: "test"}}
		
		var wg sync.WaitGroup
		errors := make(chan error, 100)
		
		for i := 0; i < 100; i++ {
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
		errors := make(chan error, 100)
		
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var u Union2[A, B]
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
}