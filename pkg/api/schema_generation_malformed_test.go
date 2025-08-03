package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

// Test case for reproducing the malformed schema issue
// AdminUserResponse and UserResponse should not have 'Body' field in their OpenAPI schema
// when used as elements in a Union type within a response Body

type MalformedTestAdminUserResponse struct {
	UserID    string `gork:"userID"`
	Username  string `gork:"username"`
	CreatedAt string `gork:"createdAt"`
	UpdatedAt string `gork:"updatedAt"`
}

type MalformedTestUserResponse struct {
	UserID   string `gork:"userID"`
	Username string `gork:"username"`
}

type MalformedTestListUsersResponse struct {
	Body unions.Union2[[]MalformedTestAdminUserResponse, []MalformedTestUserResponse]
}

func TestSchemaGenerationDoesNotIncludeBodyFieldForUnionElements(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Generate schema for AdminUserResponse (used as element in Union)
	adminUserType := reflect.TypeOf(MalformedTestAdminUserResponse{})
	adminSchema := generator.generateSchemaFromType(adminUserType, "", components)

	// The schema generation now creates component references for struct types
	// So we should get a reference schema pointing to the component
	if adminSchema.Ref == "" {
		t.Errorf("Expected AdminUserResponse schema to be a component reference, got: %+v", adminSchema)
	}

	expectedRef := "#/components/schemas/MalformedTestAdminUserResponse"
	if adminSchema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, adminSchema.Ref)
	}

	// Check that the component schema was created
	componentSchema, exists := components.Schemas["MalformedTestAdminUserResponse"]
	if !exists {
		t.Fatal("Expected MalformedTestAdminUserResponse component to be created")
	}

	// The component schema should NOT contain a 'Body' field
	if _, hasBodyField := componentSchema.Properties["Body"]; hasBodyField {
		t.Errorf("AdminUserResponse component schema incorrectly contains 'Body' field. Properties: %+v", componentSchema.Properties)
	}

	// The component schema should contain the actual fields directly
	if componentSchema.Properties == nil {
		t.Fatalf("AdminUserResponse component schema should have properties. Got schema: %+v", componentSchema)
	}

	expectedFields := []string{"userID", "username", "createdAt", "updatedAt"}
	for _, field := range expectedFields {
		if _, exists := componentSchema.Properties[field]; !exists {
			t.Errorf("AdminUserResponse component schema missing field '%s'. Properties: %+v", field, componentSchema.Properties)
		}
	}

	// Generate schema for UserResponse (used as element in Union)
	userType := reflect.TypeOf(MalformedTestUserResponse{})
	userSchema := generator.generateSchemaFromType(userType, "", components)

	// Should also be a component reference
	if userSchema.Ref == "" {
		t.Errorf("Expected UserResponse schema to be a component reference, got: %+v", userSchema)
	}

	expectedUserRef := "#/components/schemas/MalformedTestUserResponse"
	if userSchema.Ref != expectedUserRef {
		t.Errorf("Expected ref %q, got %q", expectedUserRef, userSchema.Ref)
	}

	// Check that the component schema was created
	userComponentSchema, exists := components.Schemas["MalformedTestUserResponse"]
	if !exists {
		t.Fatal("Expected MalformedTestUserResponse component to be created")
	}

	// The component schema should NOT contain a 'Body' field
	if _, hasBodyField := userComponentSchema.Properties["Body"]; hasBodyField {
		t.Errorf("UserResponse component schema incorrectly contains 'Body' field. Properties: %+v", userComponentSchema.Properties)
	}

	// The component schema should contain the actual fields directly
	if userComponentSchema.Properties == nil {
		t.Fatal("UserResponse component schema should have properties")
	}

	expectedUserFields := []string{"userID", "username"}
	for _, field := range expectedUserFields {
		if _, exists := userComponentSchema.Properties[field]; !exists {
			t.Errorf("UserResponse component schema missing field '%s'. Properties: %+v", field, userComponentSchema.Properties)
		}
	}
}

func TestUnionResponseSchemaGeneration(t *testing.T) {
	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// Generate schema for the response that contains the Union in its Body
	responseType := reflect.TypeOf(MalformedTestListUsersResponse{})
	responseSchema := generator.generateSchemaFromType(responseType, "", components)

	// The schema generation now creates component references for struct types
	// So we should get a reference schema pointing to the component
	if responseSchema.Ref == "" {
		t.Errorf("Expected ListUsersResponse schema to be a component reference, got: %+v", responseSchema)
	}

	expectedRef := "#/components/schemas/MalformedTestListUsersResponse"
	if responseSchema.Ref != expectedRef {
		t.Errorf("Expected ref %q, got %q", expectedRef, responseSchema.Ref)
	}

	// Check that the component schema was created
	componentSchema, exists := components.Schemas["MalformedTestListUsersResponse"]
	if !exists {
		t.Fatal("Expected MalformedTestListUsersResponse component to be created")
	}

	// The component schema should have a Body field
	if componentSchema.Properties == nil {
		t.Fatal("ListUsersResponse component schema should have properties")
	}

	bodyProperty, hasBody := componentSchema.Properties["Body"]
	if !hasBody {
		t.Fatal("ListUsersResponse component schema should have a 'Body' field")
	}

	// The Body field should be a Union (oneOf) - either directly or as a reference
	if len(bodyProperty.OneOf) == 2 {
		// Direct oneOf schema - this is fine
	} else if bodyProperty.Ref != "" {
		// Reference to a union component schema
		// Extract the component name from the reference
		refParts := strings.Split(bodyProperty.Ref, "/")
		if len(refParts) > 0 {
			componentName := refParts[len(refParts)-1]
			unionSchema, exists := components.Schemas[componentName]
			if !exists {
				t.Fatalf("Union component schema %s not found", componentName)
			}

			// Check if the union component has oneOf
			if len(unionSchema.OneOf) != 2 {
				t.Errorf("Union component schema should have oneOf with 2 options, got: %+v", unionSchema.OneOf)
			}
		}
	} else {
		t.Errorf("Body field should be either a direct oneOf or reference to union component, got: %+v", bodyProperty)
	}
}

// Test that reproduces the actual panic from the OpenAPI generation
func TestHandlerResponseWithoutConventionSectionsCausesPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// We expect this to panic with a message containing the expected text
			panicMsg := r.(string)
			expectedText := "response type must use Convention Over Configuration sections (Body, Headers, Cookies)"
			if !strings.Contains(panicMsg, expectedText) {
				t.Errorf("Expected panic message to contain %q, got %q", expectedText, panicMsg)
			}
		} else {
			t.Error("Expected panic but didn't get one")
		}
	}()

	generator := &ConventionOpenAPIGenerator{}
	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	// This should panic because MalformedTestUserResponse doesn't have Body/Headers/Cookies sections
	userResponseType := reflect.TypeOf((*MalformedTestUserResponse)(nil))
	operation := &Operation{}
	mockRoute := &RouteInfo{
		Method:      "GET",
		Path:        "/test",
		HandlerName: "TestHandler",
	}
	generator.processResponseSections(userResponseType, operation, components, mockRoute)
}
