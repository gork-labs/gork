package unions

import (
	"encoding/json"
	"testing"
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