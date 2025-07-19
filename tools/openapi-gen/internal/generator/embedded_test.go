package generator

import (
	"encoding/json"
	"testing"
)

func TestEmbeddedStructsInSchema(t *testing.T) {
	tests := []struct {
		name           string
		extractedType  ExtractedType
		expectedFields []string
	}{
		{
			name: "Simple embedded struct",
			extractedType: ExtractedType{
				Name:    "ExtendedUser",
				Package: "models",
				Fields: []ExtractedField{
					{Name: "CreatedAt", JSONTag: "createdAt", Type: "string"},
				},
				EmbeddedTypes: []string{"BaseUser"},
			},
			expectedFields: []string{"createdAt", "id", "name"},
		},
		{
			name: "Multiple embedded structs",
			extractedType: ExtractedType{
				Name:    "AdminUser",
				Package: "models",
				Fields: []ExtractedField{
					{Name: "AdminLevel", JSONTag: "adminLevel", Type: "int"},
				},
				EmbeddedTypes: []string{"BaseUser", "Timestamps"},
			},
			expectedFields: []string{"adminLevel", "id", "name", "createdAt", "updatedAt"},
		},
		{
			name: "Embedded struct with no JSON tags (union options)",
			extractedType: ExtractedType{
				Name:    "ExtendedOptions",
				Package: "models",
				Fields: []ExtractedField{
					{Name: "ExtraField", Type: "string", IsPointer: true},
				},
				EmbeddedTypes: []string{"BaseOptions"},
			},
			expectedFields: []string{"extrafield", "option1", "option2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create generator
			g := New("Test API", "1.0.0")

			// Add base types to typeMap
			g.typeMap["BaseUser"] = ExtractedType{
				Name:    "BaseUser",
				Package: "models",
				Fields: []ExtractedField{
					{Name: "ID", JSONTag: "id", Type: "string"},
					{Name: "Name", JSONTag: "name", Type: "string"},
				},
			}

			g.typeMap["Timestamps"] = ExtractedType{
				Name:    "Timestamps",
				Package: "models",
				Fields: []ExtractedField{
					{Name: "CreatedAt", JSONTag: "createdAt", Type: "string"},
					{Name: "UpdatedAt", JSONTag: "updatedAt", Type: "string"},
				},
			}

			g.typeMap["BaseOptions"] = ExtractedType{
				Name:    "BaseOptions",
				Package: "models",
				Fields: []ExtractedField{
					{Name: "Option1", Type: "string", IsPointer: true},
					{Name: "Option2", Type: "string", IsPointer: true},
				},
			}

			// Generate schema
			g.generateSchema(tt.extractedType)

			// Check if schema was created
			schema, exists := g.spec.Components.Schemas[tt.extractedType.Name]
			if !exists {
				t.Fatalf("Schema for %s was not created", tt.extractedType.Name)
			}

			// Check if all expected fields are present
			for _, expectedField := range tt.expectedFields {
				if _, ok := schema.Properties[expectedField]; !ok {
					t.Errorf("Expected field %s not found in schema properties", expectedField)
				}
			}

			// Check field count
			if len(schema.Properties) != len(tt.expectedFields) {
				t.Errorf("Expected %d fields, got %d", len(tt.expectedFields), len(schema.Properties))
			}
		})
	}
}

func TestEmbeddedStructsInUnions(t *testing.T) {
	// Create generator
	g := New("Test API", "1.0.0")

	// Add types to typeMap
	g.typeMap["UserResponse"] = ExtractedType{
		Name:    "UserResponse",
		Package: "handlers",
		Fields: []ExtractedField{
			{Name: "UserId", JSONTag: "userId", Type: "string"},
			{Name: "Username", JSONTag: "username", Type: "string"},
		},
	}

	g.typeMap["AdminUserResponse"] = ExtractedType{
		Name:    "AdminUserResponse",
		Package: "handlers",
		Fields: []ExtractedField{
			{Name: "CreatedAt", JSONTag: "createdAt", Type: "string"},
			{Name: "UpdatedAt", JSONTag: "updatedAt", Type: "string"},
		},
		EmbeddedTypes: []string{"UserResponse"},
	}

	// Generate schemas
	g.generateSchema(g.typeMap["UserResponse"])
	g.generateSchema(g.typeMap["AdminUserResponse"])

	// Check AdminUserResponse schema
	adminSchema, exists := g.spec.Components.Schemas["AdminUserResponse"]
	if !exists {
		t.Fatal("AdminUserResponse schema was not created")
	}

	// Verify it has all fields (embedded + own)
	expectedFields := map[string]bool{
		"userId":    true,
		"username":  true,
		"createdAt": true,
		"updatedAt": true,
	}

	for field := range expectedFields {
		if _, ok := adminSchema.Properties[field]; !ok {
			t.Errorf("Expected field %s not found in AdminUserResponse schema", field)
		}
	}

	// Verify the schema can be properly serialized
	schemaJSON, err := json.MarshalIndent(adminSchema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Basic check that the JSON contains expected fields
	jsonStr := string(schemaJSON)
	for field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("Field %s not found in serialized schema JSON", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}