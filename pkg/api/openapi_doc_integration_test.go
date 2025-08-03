package api

import (
	"testing"
)

// MockDocExtractor for testing enrichFromEmbeddedTypes
type MockDocExtractor struct {
	typeNames []string
	typeDocs  map[string]Documentation
}

func (m *MockDocExtractor) GetAllTypeNames() []string {
	return m.typeNames
}

func (m *MockDocExtractor) ExtractTypeDoc(typeName string) Documentation {
	if doc, ok := m.typeDocs[typeName]; ok {
		return doc
	}
	return Documentation{}
}

func TestEnrichFromEmbeddedTypes(t *testing.T) {
	t.Run("schema with no properties", func(t *testing.T) {
		schema := &Schema{}
		extractor := NewDocExtractor()

		// Should not panic and should return early
		enrichFromEmbeddedTypes(schema, extractor)
	})

	t.Run("schema with properties but no matching types", func(t *testing.T) {
		schema := &Schema{
			Properties: map[string]*Schema{
				"field1": {Type: "string"},
				"field2": {Type: "string"},
			},
		}
		extractor := NewDocExtractor()

		// Add documentation for a type that doesn't match our fields
		extractor.docs = map[string]Documentation{
			"SomeOtherType": {
				Fields: map[string]FieldDoc{
					"otherField": {Description: "Other field"},
				},
			},
		}

		enrichFromEmbeddedTypes(schema, extractor)

		// Properties should still have no descriptions
		if schema.Properties["field1"].Description != "" {
			t.Error("Expected field1 to have no description")
		}
		if schema.Properties["field2"].Description != "" {
			t.Error("Expected field2 to have no description")
		}
	})

	t.Run("schema with properties and matching type", func(t *testing.T) {
		schema := &Schema{
			Properties: map[string]*Schema{
				"userID":   {Type: "string"},
				"username": {Type: "string"},
			},
		}
		extractor := NewDocExtractor()

		// Add documentation for UserType that matches our fields
		extractor.docs = map[string]Documentation{
			"UserType": {
				Fields: map[string]FieldDoc{
					"userID":   {Description: "ID of the user"},
					"username": {Description: "Username of the user"},
				},
			},
		}

		enrichFromEmbeddedTypes(schema, extractor)

		// Properties should now have descriptions
		if schema.Properties["userID"].Description != "ID of the user" {
			t.Errorf("Expected userID description 'ID of the user', got '%s'", schema.Properties["userID"].Description)
		}
		if schema.Properties["username"].Description != "Username of the user" {
			t.Errorf("Expected username description 'Username of the user', got '%s'", schema.Properties["username"].Description)
		}
	})

	t.Run("schema with some properties already having descriptions", func(t *testing.T) {
		schema := &Schema{
			Properties: map[string]*Schema{
				"userID":   {Type: "string", Description: "Existing description"},
				"username": {Type: "string"},
			},
		}
		extractor := NewDocExtractor()

		// Add documentation for UserType
		extractor.docs = map[string]Documentation{
			"UserType": {
				Fields: map[string]FieldDoc{
					"userID":   {Description: "ID of the user"},
					"username": {Description: "Username of the user"},
				},
			},
		}

		enrichFromEmbeddedTypes(schema, extractor)

		// userID should keep existing description, username should get new one
		if schema.Properties["userID"].Description != "Existing description" {
			t.Errorf("Expected userID to keep existing description, got '%s'", schema.Properties["userID"].Description)
		}
		if schema.Properties["username"].Description != "Username of the user" {
			t.Errorf("Expected username description 'Username of the user', got '%s'", schema.Properties["username"].Description)
		}
	})

	t.Run("empty extractor with no types", func(t *testing.T) {
		schema := &Schema{
			Properties: map[string]*Schema{
				"field1": {Type: "string"},
			},
		}
		extractor := NewDocExtractor()
		// Empty extractor - no types documented

		enrichFromEmbeddedTypes(schema, extractor)

		// Should not panic and should not change anything
		if schema.Properties["field1"].Description != "" {
			t.Error("Expected field1 to have no description")
		}
	})

	t.Run("type with no field documentation", func(t *testing.T) {
		schema := &Schema{
			Properties: map[string]*Schema{
				"field1": {Type: "string"},
			},
		}
		extractor := NewDocExtractor()

		// Add a type that has no field documentation
		extractor.docs = map[string]Documentation{
			"TypeWithoutFields": {
				Description: "A type without field docs",
				Fields:      map[string]FieldDoc{}, // Empty fields map
			},
		}

		enrichFromEmbeddedTypes(schema, extractor)

		// Should not change anything since type has no fields
		if schema.Properties["field1"].Description != "" {
			t.Error("Expected field1 to have no description")
		}
	})

	t.Run("extractor returns type with zero fields triggering continue", func(t *testing.T) {
		schema := &Schema{
			Properties: map[string]*Schema{
				"field1": {Type: "string"},
			},
		}
		extractor := NewDocExtractor()

		// Add a type to the extractor's internal registry but with empty fields
		// This simulates ExtractTypeDoc returning a type with len(typeDoc.Fields) == 0
		extractor.docs = map[string]Documentation{
			"EmptyFieldsType": {
				Description: "Type with empty fields",
				Fields:      map[string]FieldDoc{}, // This triggers continue in enrichFromEmbeddedTypes
			},
		}

		enrichFromEmbeddedTypes(schema, extractor)

		// Should not change field1 since the type has no field documentation to provide
		if schema.Properties["field1"].Description != "" {
			t.Error("Expected field1 to remain without description")
		}
	})

	t.Run("extractor with type that gets filtered by continue", func(t *testing.T) {
		schema := &Schema{
			Properties: map[string]*Schema{
				"field1": {Type: "string"},
			},
		}

		// Create a mock extractor that returns a type with no fields, triggering the continue
		mockExtractor := &MockDocExtractor{
			typeNames: []string{"TypeWithoutFields", "TypeWithFields"},
			typeDocs: map[string]Documentation{
				"TypeWithoutFields": {
					Description: "Type description",
					Fields:      nil, // This will cause len(typeDoc.Fields) == 0, triggering continue
				},
				"TypeWithFields": {
					Description: "Type with fields",
					Fields: map[string]FieldDoc{
						"field1": {Description: "Field description"},
					},
				},
			},
		}

		enrichFromEmbeddedTypes(schema, mockExtractor)

		// Should get description from TypeWithFields since TypeWithoutFields is skipped by continue
		if schema.Properties["field1"].Description != "Field description" {
			t.Errorf("Expected field1 to get description from TypeWithFields, got '%s'", schema.Properties["field1"].Description)
		}
	})
}

func TestEnrichFromContextualRequestType(t *testing.T) {
	tests := []struct {
		name       string
		schema     *Schema
		schemaName string
		extractor  *MockDocExtractor
		expectDocs bool
	}{
		{
			name:       "nil schema",
			schema:     nil,
			schemaName: "UpdateUserBody",
			extractor:  &MockDocExtractor{},
			expectDocs: false,
		},
		{
			name: "schema with no properties",
			schema: &Schema{
				Properties: nil,
			},
			schemaName: "UpdateUserBody",
			extractor:  &MockDocExtractor{},
			expectDocs: false,
		},
		{
			name: "schema with properties that already have descriptions",
			schema: &Schema{
				Properties: map[string]*Schema{
					"username": {Description: "Already documented"},
				},
			},
			schemaName: "UpdateUserBody",
			extractor:  &MockDocExtractor{},
			expectDocs: false,
		},
		{
			name: "schema with properties needing docs - preferred type found",
			schema: &Schema{
				Properties: map[string]*Schema{
					"username": {Description: ""},
					"userID":   {Description: ""},
				},
			},
			schemaName: "UpdateUserBody",
			extractor: &MockDocExtractor{
				typeNames: []string{"UpdateUserRequest"},
				typeDocs: map[string]Documentation{
					"UpdateUserRequest": {
						Fields: map[string]FieldDoc{
							"username": {Description: "Updated username"},
							"userID":   {Description: "User ID to update"},
						},
					},
				},
			},
			expectDocs: true,
		},
		{
			name: "schema with properties needing docs - no preferred type",
			schema: &Schema{
				Properties: map[string]*Schema{
					"username": {Description: ""},
				},
			},
			schemaName: "UpdateUserBody",
			extractor: &MockDocExtractor{
				typeNames: []string{"OtherType"},
				typeDocs: map[string]Documentation{
					"OtherType": {
						Fields: map[string]FieldDoc{
							"username": {Description: "Generic username"},
						},
					},
				},
			},
			expectDocs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalProperties := map[string]*Schema{}
			if tt.schema != nil && tt.schema.Properties != nil {
				for k, v := range tt.schema.Properties {
					originalProperties[k] = &Schema{Description: v.Description}
				}
			}

			enrichFromContextualRequestType(tt.schema, tt.schemaName, tt.extractor)

			if tt.expectDocs && tt.schema != nil {
				for propName, propSchema := range tt.schema.Properties {
					if originalProperties[propName].Description == "" {
						// If the original was empty, it should now have documentation
						if propSchema.Description == "" {
							t.Errorf("Expected property %s to have documentation, but it was empty", propName)
						}
					}
				}
			}
		})
	}
}

func TestGetRequestTypeNameFromContextualSchema(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		expected   string
	}{
		{
			name:       "Body schema",
			schemaName: "UpdateUserBody",
			expected:   "UpdateUserRequest",
		},
		{
			name:       "Headers schema",
			schemaName: "CreateUserHeaders",
			expected:   "CreateUserRequest",
		},
		{
			name:       "Query schema",
			schemaName: "ListUsersQuery",
			expected:   "ListUsersRequest",
		},
		{
			name:       "Path schema",
			schemaName: "GetUserPath",
			expected:   "GetUserRequest",
		},
		{
			name:       "Cookies schema",
			schemaName: "DeleteUserCookies",
			expected:   "DeleteUserRequest",
		},
		{
			name:       "Response schema",
			schemaName: "UserResponse",
			expected:   "UserResponse",
		},
		{
			name:       "Non-contextual schema",
			schemaName: "RegularType",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRequestTypeNameFromContextualSchema(tt.schemaName)
			if result != tt.expected {
				t.Errorf("getRequestTypeNameFromContextualSchema(%q) = %q, want %q", tt.schemaName, result, tt.expected)
			}
		})
	}
}

func TestEnrichPropertiesWithPreferredType(t *testing.T) {
	tests := []struct {
		name             string
		propsNeedingDocs map[string]*Schema
		preferredType    string
		extractor        *MockDocExtractor
		expectDocs       bool
	}{
		{
			name: "no preferred type - falls back to general search",
			propsNeedingDocs: map[string]*Schema{
				"username": {Description: ""},
			},
			preferredType: "",
			extractor: &MockDocExtractor{
				typeNames: []string{"SomeType"},
				typeDocs: map[string]Documentation{
					"SomeType": {
						Fields: map[string]FieldDoc{
							"username": {Description: "General username"},
						},
					},
				},
			},
			expectDocs: true,
		},
		{
			name: "preferred type with documentation",
			propsNeedingDocs: map[string]*Schema{
				"username": {Description: ""},
				"userID":   {Description: ""},
			},
			preferredType: "UpdateUserRequest",
			extractor: &MockDocExtractor{
				typeNames: []string{"UpdateUserRequest", "OtherType"},
				typeDocs: map[string]Documentation{
					"UpdateUserRequest": {
						Fields: map[string]FieldDoc{
							"username": {Description: "Preferred username"},
							"userID":   {Description: "Preferred user ID"},
						},
					},
					"OtherType": {
						Fields: map[string]FieldDoc{
							"username": {Description: "Other username"},
						},
					},
				},
			},
			expectDocs: true,
		},
		{
			name: "preferred type with partial documentation",
			propsNeedingDocs: map[string]*Schema{
				"username": {Description: ""},
				"email":    {Description: ""},
			},
			preferredType: "UpdateUserRequest",
			extractor: &MockDocExtractor{
				typeNames: []string{"UpdateUserRequest", "OtherType"},
				typeDocs: map[string]Documentation{
					"UpdateUserRequest": {
						Fields: map[string]FieldDoc{
							"username": {Description: "Preferred username"},
							// email is not in preferred type
						},
					},
					"OtherType": {
						Fields: map[string]FieldDoc{
							"email": {Description: "Email from other type"},
						},
					},
				},
			},
			expectDocs: true,
		},
		{
			name: "preferred type with no documentation",
			propsNeedingDocs: map[string]*Schema{
				"username": {Description: ""},
			},
			preferredType: "UpdateUserRequest",
			extractor: &MockDocExtractor{
				typeNames: []string{"UpdateUserRequest", "OtherType"},
				typeDocs: map[string]Documentation{
					"UpdateUserRequest": {
						Fields: map[string]FieldDoc{},
					},
					"OtherType": {
						Fields: map[string]FieldDoc{
							"username": {Description: "Fallback username"},
						},
					},
				},
			},
			expectDocs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to check which properties were documented
			originalDescs := make(map[string]string)
			for propName, propSchema := range tt.propsNeedingDocs {
				originalDescs[propName] = propSchema.Description
			}

			enrichPropertiesWithPreferredType(tt.propsNeedingDocs, tt.preferredType, tt.extractor)

			if tt.expectDocs {
				foundDoc := false
				// Check the actual property schemas for applied documentation
				for propName, propSchema := range tt.propsNeedingDocs {
					if originalDescs[propName] == "" && propSchema.Description != "" {
						foundDoc = true
						break
					}
				}

				// Also check if any properties were completely removed from the map
				// (which happens when they are fully documented by the preferred type)
				if len(tt.propsNeedingDocs) < len(originalDescs) {
					foundDoc = true
				}

				if !foundDoc {
					t.Errorf("Expected to find documentation but none was applied. Props remaining: %d, original: %d",
						len(tt.propsNeedingDocs), len(originalDescs))
				}
			}
		})
	}
}

func TestIsContextualSchemaName(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		expected   bool
	}{
		{
			name:       "Body schema",
			schemaName: "UpdateUserBody",
			expected:   true,
		},
		{
			name:       "Headers schema",
			schemaName: "RequestHeaders",
			expected:   true,
		},
		{
			name:       "Query schema",
			schemaName: "SearchQuery",
			expected:   true,
		},
		{
			name:       "Path schema",
			schemaName: "UserPath",
			expected:   true,
		},
		{
			name:       "Cookies schema",
			schemaName: "SessionCookies",
			expected:   true,
		},
		{
			name:       "Response schema",
			schemaName: "UserResponse",
			expected:   true,
		},
		{
			name:       "Regular type",
			schemaName: "RegularType",
			expected:   false,
		},
		{
			name:       "Empty string",
			schemaName: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isContextualSchemaName(tt.schemaName)
			if result != tt.expected {
				t.Errorf("isContextualSchemaName(%q) = %v, want %v", tt.schemaName, result, tt.expected)
			}
		})
	}
}

// TestContextualSchemaDocumentationIntegration tests the complete flow of contextual schema documentation
// extraction for the specific case that was originally broken: nested anonymous struct fields.
func TestContextualSchemaDocumentationIntegration(t *testing.T) {
	// This test covers the exact scenario that was reported:
	// UpdateUserPreferencesBody schema should get field documentation from UpdateUserPreferencesRequest
	// which contains nested anonymous struct fields with comments

	extractor := &MockDocExtractor{
		typeNames: []string{"UpdateUserPreferencesRequest"},
		typeDocs: map[string]Documentation{
			"UpdateUserPreferencesRequest": {
				Description: "UpdateUserPreferencesRequest represents the request for updating user preferences.",
				Fields: map[string]FieldDoc{
					// Both Go field names and gork tag names (as extracted by real DocExtractor)
					"PaymentMethod":              {Description: "PaymentMethod contains the user's payment method"},
					"paymentMethod":              {Description: "PaymentMethod contains the user's payment method"},
					"PrimaryNotificationChannel": {Description: "PrimaryNotificationChannel is the user's preferred notification channel"},
					"primaryNotificationChannel": {Description: "PrimaryNotificationChannel is the user's preferred notification channel"},
					"UserID":                     {Description: "UserID is the ID of the user whose preferences are being updated"},
					"userId":                     {Description: "UserID is the ID of the user whose preferences are being updated"},
				},
			},
		},
	}

	// Simulate the UpdateUserPreferencesBody schema that was missing field documentation
	schema := &Schema{
		Properties: map[string]*Schema{
			"paymentMethod": {
				Description: "", // Empty - should get documented
			},
			"primaryNotificationChannel": {
				Description: "", // Empty - should get documented (this was the main issue)
			},
		},
	}

	// Apply the contextual enrichment (this is what happens in enrichSchemaWithTypeDoc)
	enrichFromContextualRequestType(schema, "UpdateUserPreferencesBody", extractor)

	// Verify that both fields got proper documentation
	paymentMethodDesc := schema.Properties["paymentMethod"].Description
	if paymentMethodDesc != "PaymentMethod contains the user's payment method" {
		t.Errorf("Expected paymentMethod to be documented, got: %q", paymentMethodDesc)
	}

	notificationChannelDesc := schema.Properties["primaryNotificationChannel"].Description
	if notificationChannelDesc != "PrimaryNotificationChannel is the user's preferred notification channel" {
		t.Errorf("Expected primaryNotificationChannel to be documented, got: %q", notificationChannelDesc)
	}

	t.Logf("✅ Integration test passed - both fields properly documented:")
	t.Logf("  paymentMethod: %s", paymentMethodDesc)
	t.Logf("  primaryNotificationChannel: %s", notificationChannelDesc)
}

// TestContextualSchemaTypesDocumentation tests that all contextual schema types
// (Body, Headers, Query, Path, Cookies) properly map to their request types.
func TestContextualSchemaTypesDocumentation(t *testing.T) {
	extractor := &MockDocExtractor{
		typeNames: []string{"CreateUserRequest", "ListUsersRequest", "GetUserRequest", "UpdateUserRequest", "DeleteUserRequest"},
		typeDocs: map[string]Documentation{
			"CreateUserRequest": {
				Fields: map[string]FieldDoc{
					"username": {Description: "Username for the new user"},
				},
			},
			"ListUsersRequest": {
				Fields: map[string]FieldDoc{
					"limit":  {Description: "Maximum number of users to return"},
					"offset": {Description: "Number of users to skip"},
				},
			},
			"GetUserRequest": {
				Fields: map[string]FieldDoc{
					"userId": {Description: "ID of the user to retrieve"},
				},
			},
			"UpdateUserRequest": {
				Fields: map[string]FieldDoc{
					"version": {Description: "User version for concurrency control"},
				},
			},
			"DeleteUserRequest": {
				Fields: map[string]FieldDoc{
					"session": {Description: "Session token for authentication"},
				},
			},
		},
	}

	testCases := []struct {
		contextualSchemaName string
		propertyName         string
		expectedDescription  string
	}{
		{"CreateUserBody", "username", "Username for the new user"},
		{"ListUsersQuery", "limit", "Maximum number of users to return"},
		{"ListUsersQuery", "offset", "Number of users to skip"},
		{"GetUserPath", "userId", "ID of the user to retrieve"},
		{"UpdateUserHeaders", "version", "User version for concurrency control"},
		{"DeleteUserCookies", "session", "Session token for authentication"},
	}

	for _, tc := range testCases {
		t.Run(tc.contextualSchemaName, func(t *testing.T) {
			schema := &Schema{
				Properties: map[string]*Schema{
					tc.propertyName: {Description: ""}, // Empty - should get documented
				},
			}

			enrichFromContextualRequestType(schema, tc.contextualSchemaName, extractor)

			actualDesc := schema.Properties[tc.propertyName].Description
			if actualDesc != tc.expectedDescription {
				t.Errorf("Expected %s.%s to be documented with %q, got %q",
					tc.contextualSchemaName, tc.propertyName, tc.expectedDescription, actualDesc)
			}
		})
	}
}

func TestTryPreferredTypeEnrichment(t *testing.T) {
	tests := []struct {
		name          string
		preferredType string
		extractor     *MockDocExtractor
		props         map[string]*Schema
		expectedOk    bool
		expectedProps int // remaining props after enrichment
	}{
		{
			name:          "empty preferred type",
			preferredType: "",
			extractor:     &MockDocExtractor{},
			props:         map[string]*Schema{"field1": {Description: ""}},
			expectedOk:    false,
			expectedProps: 1,
		},
		{
			name:          "preferred type with no fields",
			preferredType: "TestType",
			extractor: &MockDocExtractor{
				typeNames: []string{"TestType"},
				typeDocs: map[string]Documentation{
					"TestType": {Fields: map[string]FieldDoc{}},
				},
			},
			props:         map[string]*Schema{"field1": {Description: ""}},
			expectedOk:    false,
			expectedProps: 1,
		},
		{
			name:          "preferred type with fields but no matches",
			preferredType: "TestType",
			extractor: &MockDocExtractor{
				typeNames: []string{"TestType"},
				typeDocs: map[string]Documentation{
					"TestType": {
						Fields: map[string]FieldDoc{
							"differentField": {Description: "Different field docs"},
						},
					},
				},
			},
			props:         map[string]*Schema{"field1": {Description: ""}},
			expectedOk:    false,
			expectedProps: 1,
		},
		{
			name:          "successful enrichment - all props documented",
			preferredType: "TestType",
			extractor: &MockDocExtractor{
				typeNames: []string{"TestType"},
				typeDocs: map[string]Documentation{
					"TestType": {
						Fields: map[string]FieldDoc{
							"field1": {Description: "Field 1 docs"},
						},
					},
				},
			},
			props:         map[string]*Schema{"field1": {Description: ""}},
			expectedOk:    true,
			expectedProps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tryPreferredTypeEnrichment(tt.props, tt.preferredType, tt.extractor)
			if result != tt.expectedOk {
				t.Errorf("Expected %v, got %v", tt.expectedOk, result)
			}
			if len(tt.props) != tt.expectedProps {
				t.Errorf("Expected %d remaining props, got %d", tt.expectedProps, len(tt.props))
			}
		})
	}
}

func TestRemoveDocumentedProperties(t *testing.T) {
	props := map[string]*Schema{
		"documented":   {Description: ""},
		"undocumented": {Description: ""},
	}

	typeDoc := Documentation{
		Fields: map[string]FieldDoc{
			"documented": {Description: "Some docs"},
		},
	}

	removeDocumentedProperties(props, typeDoc)

	if len(props) != 1 {
		t.Errorf("Expected 1 remaining property, got %d", len(props))
	}

	if _, exists := props["undocumented"]; !exists {
		t.Error("Expected undocumented property to remain")
	}

	if _, exists := props["documented"]; exists {
		t.Error("Expected documented property to be removed")
	}
}

func TestTryRemainingTypesEnrichment(t *testing.T) {
	props := map[string]*Schema{
		"field1": {Description: ""},
		"field2": {Description: ""},
	}

	extractor := &MockDocExtractor{
		typeNames: []string{"PreferredType", "OtherType1", "OtherType2"},
		typeDocs: map[string]Documentation{
			"PreferredType": {Fields: map[string]FieldDoc{}}, // Empty - already tried
			"OtherType1":    {Fields: map[string]FieldDoc{}}, // Empty - skip
			"OtherType2": {
				Fields: map[string]FieldDoc{
					"field1": {Description: "Field 1 from other type"},
					"field2": {Description: "Field 2 from other type"},
				},
			},
		},
	}

	tryRemainingTypesEnrichment(props, "PreferredType", extractor)

	// Should find documentation from OtherType2
	if props["field1"].Description != "Field 1 from other type" {
		t.Error("Expected field1 to be documented")
	}
	if props["field2"].Description != "Field 2 from other type" {
		t.Error("Expected field2 to be documented")
	}
}
