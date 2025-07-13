package handlers

import (
	"encoding/json"
	"testing"
)

func TestAnyOfWithoutWrapperReq(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantOpt int // 1 for Option1, 2 for Option2
		wantErr bool
	}{
		{
			name:    "option1 by unique field",
			json:    `{"option1Field": "test1"}`,
			wantOpt: 1,
		},
		{
			name:    "option2 by unique field",
			json:    `{"option2Field": "test2"}`,
			wantOpt: 2,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req AnyOfWithoutWrapperReq
			err := json.Unmarshal([]byte(tt.json), &req)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				switch tt.wantOpt {
				case 1:
					if req.Option1 == nil {
						t.Error("Expected Option1 to be set")
					}
					if req.Option1.Option1Field != "test1" {
						t.Errorf("Option1Field = %v, want test1", req.Option1.Option1Field)
					}
				case 2:
					if req.Option2 == nil {
						t.Error("Expected Option2 to be set")
					}
					if req.Option2.Option2Field != "test2" {
						t.Errorf("Option2Field = %v, want test2", req.Option2.Option2Field)
					}
				}
			}
		})
	}
}

func TestAnyOfWithoutWrapperReq_DirectAccess(t *testing.T) {
	// Test direct field access
	req := AnyOfWithoutWrapperReq{}
	req.Option1 = &Option1{Option1Field: "direct"}

	// Should be able to access fields directly
	if req.Option1 == nil || req.Option1.Option1Field != "direct" {
		t.Error("Direct field access failed")
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{"option1Field":"direct"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}