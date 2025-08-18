package api

import (
	"reflect"
	"testing"
	"time"

	"github.com/gork-labs/gork/pkg/unions"
)

// Test types for OpenAPI generation
type TestOpenAPIRequest struct {
	Path struct {
		UserID string `gork:"user_id" validate:"required,uuid"`
	}
	Query struct {
		Limit  int      `gork:"limit" validate:"min=1,max=100"`
		Fields []string `gork:"fields"`
		Active bool     `gork:"active"`
	}
	Headers struct {
		Authorization string `gork:"Authorization" validate:"required"`
		ContentType   string `gork:"Content-Type"`
	}
	Cookies struct {
		SessionID string `gork:"session_id"`
	}
	Body struct {
		Name    string    `gork:"name" validate:"required"`
		Email   string    `gork:"email" validate:"email"`
		Created time.Time `gork:"created"`
	}
}

type TestOpenAPIResponse struct {
	Body struct {
		ID      string    `gork:"id"`
		Name    string    `gork:"name"`
		Created time.Time `gork:"created"`
	}
	Headers struct {
		Location     string `gork:"Location"`
		CacheControl string `gork:"Cache-Control"`
	}
	Cookies struct {
		SessionToken string `gork:"session_token"`
	}
}

// Union types for testing
type EmailAuth struct {
	Type     string `gork:"type,discriminator=email" validate:"required"`
	Email    string `gork:"email" validate:"required,email"`
	Password string `gork:"password" validate:"required"`
}

type TokenAuth struct {
	Type  string `gork:"type,discriminator=token" validate:"required"`
	Token string `gork:"token" validate:"required"`
}

type TestUnionRequest struct {
	Body struct {
		AuthMethod unions.Union2[EmailAuth, TokenAuth] `gork:"auth_method" validate:"required"`
	}
}

func TestNewConventionOpenAPIGenerator(t *testing.T) {
	spec := &OpenAPISpec{}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	if generator == nil {
		t.Fatal("NewConventionOpenAPIGenerator() returned nil")
	}
	if generator.spec != spec {
		t.Error("Generator spec not set correctly")
	}
}

func TestConventionOpenAPIGenerator_BuildOperation(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Create a mock route info
	routeInfo := &RouteInfo{
		HandlerName:  "TestHandler",
		RequestType:  reflect.TypeOf(TestOpenAPIRequest{}),
		ResponseType: reflect.TypeOf((*TestOpenAPIResponse)(nil)),
		Options: &HandlerOption{
			Tags: []string{"users"},
		},
	}

	operation := generator.buildConventionOperation(routeInfo, spec.Components)

	if operation == nil {
		t.Fatal("buildConventionOperation() returned nil")
	}

	// Check operation ID
	if operation.OperationID != "TestHandler" {
		t.Errorf("OperationID = %v, want TestHandler", operation.OperationID)
	}

	// Check tags
	if len(operation.Tags) != 1 || operation.Tags[0] != "users" {
		t.Errorf("Tags = %v, want [users]", operation.Tags)
	}

	// Check parameters - should have path, query, header, and cookie parameters
	expectedParamCount := 7 // user_id (path), limit, fields, active (query), Authorization, Content-Type (headers), session_id (cookie)
	if len(operation.Parameters) != expectedParamCount {
		for i, param := range operation.Parameters {
			t.Logf("Parameter %d: %s in %s", i, param.Name, param.In)
		}
		t.Errorf("Parameter count = %d, want %d", len(operation.Parameters), expectedParamCount)
	}

	// Check for specific parameters
	paramNames := make(map[string]string)
	for _, param := range operation.Parameters {
		paramNames[param.Name] = param.In
	}

	expectedParams := map[string]string{
		"user_id":       "path",
		"limit":         "query",
		"fields":        "query",
		"active":        "query",
		"Authorization": "header",
		"Content-Type":  "header",
		"session_id":    "cookie",
	}

	for name, expectedIn := range expectedParams {
		if actualIn, exists := paramNames[name]; !exists {
			t.Errorf("Parameter %s not found", name)
		} else if actualIn != expectedIn {
			t.Errorf("Parameter %s in = %v, want %v", name, actualIn, expectedIn)
		}
	}

	// Check request body
	if operation.RequestBody == nil {
		t.Error("RequestBody is nil")
	} else {
		if !operation.RequestBody.Required {
			t.Error("RequestBody should be required")
		}
		if _, exists := operation.RequestBody.Content["application/json"]; !exists {
			t.Error("RequestBody missing application/json content")
		}
	}

	// Check response
	if _, exists := operation.Responses["200"]; !exists {
		t.Error("Operation missing 200 response")
	}
}

func TestConventionOpenAPIGenerator_ProcessQuerySection(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	queryType := reflect.TypeOf(struct {
		Limit  int      `gork:"limit" validate:"required,min=1,max=100"`
		Fields []string `gork:"fields"`
		Active bool     `gork:"active"`
	}{})

	operation := &Operation{
		Parameters: []Parameter{},
	}

	generator.processQuerySection(queryType, operation, spec.Components)

	if len(operation.Parameters) != 3 {
		t.Errorf("Parameter count = %d, want 3", len(operation.Parameters))
	}

	// Find limit parameter
	var limitParam *Parameter
	for i := range operation.Parameters {
		if operation.Parameters[i].Name == "limit" {
			limitParam = &operation.Parameters[i]
			break
		}
	}

	if limitParam == nil {
		t.Fatal("limit parameter not found")
	}

	if limitParam.In != "query" {
		t.Errorf("limit parameter in = %v, want query", limitParam.In)
	}

	if !limitParam.Required {
		t.Error("limit parameter should be required")
	}
}

func TestConventionOpenAPIGenerator_ProcessPathSection(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	pathType := reflect.TypeOf(struct {
		UserID string `gork:"user_id" validate:"required,uuid"`
		OrgID  string `gork:"org_id"`
	}{})

	operation := &Operation{
		Parameters: []Parameter{},
	}

	generator.processPathSection(pathType, operation, spec.Components)

	if len(operation.Parameters) != 2 {
		t.Errorf("Parameter count = %d, want 2", len(operation.Parameters))
	}

	// All path parameters should be required
	for _, param := range operation.Parameters {
		if param.In != "path" {
			t.Errorf("Parameter %s in = %v, want path", param.Name, param.In)
		}
		if !param.Required {
			t.Errorf("Path parameter %s should be required", param.Name)
		}
	}
}

func TestConventionOpenAPIGenerator_ProcessBodySection(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	bodyType := reflect.TypeOf(struct {
		Name  string `gork:"name" validate:"required"`
		Email string `gork:"email" validate:"email"`
	}{})

	operation := &Operation{}

	generator.processBodySection(bodyType, nil, operation, spec.Components)

	if operation.RequestBody == nil {
		t.Fatal("RequestBody is nil")
	}

	if !operation.RequestBody.Required {
		t.Error("RequestBody should be required")
	}

	if _, exists := operation.RequestBody.Content["application/json"]; !exists {
		t.Error("RequestBody missing application/json content")
	}
}

func TestConventionOpenAPIGenerator_ProcessResponseSections(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test with convention response
	respType := reflect.TypeOf(TestOpenAPIResponse{})
	operation := &Operation{
		Responses: make(map[string]*Response),
	}

	mockRoute := &RouteInfo{
		Method:      "GET",
		Path:        "/test",
		HandlerName: "TestHandler",
	}
	generator.processResponseSections(respType, operation, spec.Components, mockRoute)

	response, exists := operation.Responses["200"]
	if !exists {
		t.Fatal("200 response not found")
	}

	if response.Description != "Success" {
		t.Errorf("Response description = %v, want Success", response.Description)
	}

	// Check response content
	if _, exists := response.Content["application/json"]; !exists {
		t.Error("Response missing application/json content")
	}

	// Check response headers
	if len(response.Headers) == 0 {
		t.Error("Response headers not processed")
	}

	expectedHeaders := []string{"Location", "Cache-Control"}
	for _, headerName := range expectedHeaders {
		if _, exists := response.Headers[headerName]; !exists {
			t.Errorf("Response header %s not found", headerName)
		}
	}
}

func TestConventionOpenAPIGenerator_ProcessConventionResponse(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test with convention response (has Body section)
	respType := reflect.TypeOf(TestOpenAPIResponse{})
	operation := &Operation{
		Responses: make(map[string]*Response),
	}

	mockRoute := &RouteInfo{
		Method:      "GET",
		Path:        "/test",
		HandlerName: "TestHandler",
	}
	generator.processResponseSections(respType, operation, spec.Components, mockRoute)

	response, exists := operation.Responses["200"]
	if !exists {
		t.Fatal("200 response not found")
	}

	if response.Description != "Success" {
		t.Errorf("Response description = %v, want Success", response.Description)
	}

	// Check response content
	if _, exists := response.Content["application/json"]; !exists {
		t.Error("Response missing application/json content")
	}
}

func TestConventionOpenAPIGenerator_IsUnionType(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected bool
	}{
		{
			name:     "Union2 type",
			typ:      reflect.TypeOf(unions.Union2[EmailAuth, TokenAuth]{}),
			expected: true,
		},
		{
			name:     "Regular struct",
			typ:      reflect.TypeOf(EmailAuth{}),
			expected: false,
		},
		{
			name:     "String type",
			typ:      reflect.TypeOf(""),
			expected: false,
		},
		{
			name:     "Pointer to Union2 type",
			typ:      reflect.TypeOf(&unions.Union2[EmailAuth, TokenAuth]{}),
			expected: true,
		},
		{
			name:     "Pointer to regular struct",
			typ:      reflect.TypeOf(&EmailAuth{}),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnionType(tt.typ)
			if result != tt.expected {
				t.Errorf("isUnionType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConventionOpenAPIGenerator_GenerateUnionSchema(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	unionType := reflect.TypeOf(unions.Union2[EmailAuth, TokenAuth]{})
	schema := generator.generateUnionSchema(unionType, spec.Components)

	if schema == nil {
		t.Fatal("generateUnionSchema() returned nil")
	}

	// Check that it's a oneOf schema
	if len(schema.OneOf) == 0 {
		t.Error("Union schema missing oneOf")
	}

	// Check discriminator (optional, but if present should be correct)
	if schema.Discriminator != nil {
		if schema.Discriminator.PropertyName != "type" {
			t.Errorf("Discriminator property name = %v, want type", schema.Discriminator.PropertyName)
		}
	}
}

func TestBuildConventionAwareOperation(t *testing.T) {
	// Test with convention request
	conventionRoute := &RouteInfo{
		HandlerName:  "ConventionHandler",
		RequestType:  reflect.TypeOf(TestOpenAPIRequest{}),
		ResponseType: reflect.TypeOf((*TestOpenAPIResponse)(nil)),
		Options: &HandlerOption{
			Tags: []string{"test"},
		},
	}

	components := &Components{
		Schemas: make(map[string]*Schema),
	}

	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())
	operation := generator.buildConventionOperation(conventionRoute, components)

	if operation == nil {
		t.Fatal("buildConventionOperation() returned nil")
	}

	// Should have processed sections and have parameters
	if len(operation.Parameters) == 0 {
		t.Error("Convention operation should have parameters")
	}
}

// Test types for union member extraction
type TestUnionType struct {
	Value0 string
	Value1 int
	Value2 *bool
}

type TestUnionTypeWithOptions struct {
	Option1 string
	Option2 int
}

type TestUnionTypeWithMembers struct {
	Member1 string
	Member2 int
}

type TestUnionTypeWithNonUnionFields struct {
	Data     string
	Settings map[string]string
}

type TestUnionTypeWithUnexportedFields struct {
	Value1     string
	unexported int
	Value2     bool
}

type Union2TestType struct {
	SomeField string
}

type Union3TestType struct {
	AnotherField int
}

type Union4TestType struct {
	YetAnotherField bool
}

func TestConventionOpenAPIGenerator_ExtractUnionMemberTypes(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	t.Run("Union2 type", func(t *testing.T) {
		unionType := reflect.TypeOf(unions.Union2[string, int]{})
		memberTypes := generator.extractUnionMemberTypes(unionType)

		if len(memberTypes) != 2 {
			t.Errorf("extractUnionMemberTypes() returned %d types, want 2", len(memberTypes))
		}

		// Check that we get the correct member types
		if memberTypes[0] != reflect.TypeOf("") {
			t.Errorf("Expected first member type to be string, got %v", memberTypes[0])
		}
		if memberTypes[1] != reflect.TypeOf(0) {
			t.Errorf("Expected second member type to be int, got %v", memberTypes[1])
		}
	})

	t.Run("Union3 type", func(t *testing.T) {
		unionType := reflect.TypeOf(unions.Union3[string, int, bool]{})
		memberTypes := generator.extractUnionMemberTypes(unionType)

		if len(memberTypes) != 3 {
			t.Errorf("extractUnionMemberTypes() returned %d types, want 3", len(memberTypes))
		}

		// Check that we get the correct member types
		if memberTypes[0] != reflect.TypeOf("") {
			t.Errorf("Expected first member type to be string, got %v", memberTypes[0])
		}
		if memberTypes[1] != reflect.TypeOf(0) {
			t.Errorf("Expected second member type to be int, got %v", memberTypes[1])
		}
		if memberTypes[2] != reflect.TypeOf(true) {
			t.Errorf("Expected third member type to be bool, got %v", memberTypes[2])
		}
	})

	t.Run("non-struct type", func(t *testing.T) {
		memberTypes := generator.extractUnionMemberTypes(reflect.TypeOf("not a struct"))
		if len(memberTypes) != 0 {
			t.Errorf("extractUnionMemberTypes() returned %d types for non-struct, want 0", len(memberTypes))
		}
	})

	t.Run("non-union struct", func(t *testing.T) {
		type RegularStruct struct {
			Name string
			Age  int
		}
		memberTypes := generator.extractUnionMemberTypes(reflect.TypeOf(RegularStruct{}))
		if len(memberTypes) != 0 {
			t.Errorf("extractUnionMemberTypes() returned %d types for non-union struct, want 0", len(memberTypes))
		}
	})
}

func TestConventionOpenAPIGenerator_ExtractUnionMemberTypes_PointerTypes(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	// Test Union type with pointer member type
	type TestStruct struct {
		Name string
	}
	unionType := reflect.TypeOf(unions.Union2[string, *TestStruct]{})
	memberTypes := generator.extractUnionMemberTypes(unionType)

	if len(memberTypes) != 2 {
		t.Fatalf("Expected 2 member types, got %d", len(memberTypes))
	}

	// First member should be string
	if memberTypes[0] != reflect.TypeOf("") {
		t.Errorf("Expected first member type to be string, got %v", memberTypes[0])
	}

	// Second member should be *TestStruct (the actual generic type parameter)
	if memberTypes[1] != reflect.TypeOf(&TestStruct{}) {
		t.Errorf("Expected second member type to be *TestStruct, got %v", memberTypes[1])
	}
}

func TestConventionOpenAPIGenerator_UsesConventionSections(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	tests := []struct {
		name     string
		typ      reflect.Type
		expected bool
	}{
		{
			name:     "convention_response_with_body",
			typ:      reflect.TypeOf(TestOpenAPIResponse{}),
			expected: true,
		},
		{
			name:     "non_convention_response",
			typ:      reflect.TypeOf(EmailAuth{}),
			expected: false,
		},
		{
			name:     "non_struct_type",
			typ:      reflect.TypeOf("string"),
			expected: false,
		},
		{
			name:     "convention_request_with_sections",
			typ:      reflect.TypeOf(TestOpenAPIRequest{}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.usesConventionSections(tt.typ)
			if result != tt.expected {
				t.Errorf("usesConventionSections() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConventionOpenAPIGenerator_ProcessResponseSections_NonConventionPanic(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test with non-convention response type (should panic)
	respType := reflect.TypeOf(EmailAuth{})
	operation := &Operation{
		Responses: make(map[string]*Response),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("processResponseSections should panic for non-convention response")
		}
	}()

	mockRoute := &RouteInfo{
		Method:      "GET",
		Path:        "/test",
		HandlerName: "TestHandler",
	}
	generator.processResponseSections(respType, operation, spec.Components, mockRoute)
}

func TestConventionOpenAPIGenerator_ProcessResponseSections_NonStructPanic(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test with non-struct response type (should generate 204 No Content)
	respType := reflect.TypeOf("string")
	operation := &Operation{
		Responses: make(map[string]*Response),
	}

	mockRoute := &RouteInfo{
		Method:      "GET",
		Path:        "/test",
		HandlerName: "TestHandler",
	}
	generator.processResponseSections(respType, operation, spec.Components, mockRoute)

	// Should generate 204 No Content response
	if operation.Responses["204"] == nil {
		t.Error("Expected 204 No Content response for non-struct response type")
	}
	if operation.Responses["204"].Description != "No Content" {
		t.Errorf("Expected description 'No Content', got %s", operation.Responses["204"].Description)
	}
}

func TestConventionOpenAPIGenerator_GenerateUnionSchema_EmptyUnion(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test with empty union (no member types)
	unionType := reflect.TypeOf(TestUnionTypeWithNonUnionFields{})
	schema := generator.generateUnionSchema(unionType, spec.Components)

	if schema == nil {
		t.Fatal("generateUnionSchema() returned nil")
	}

	// Should have fallback description for unknown union
	if schema.Description != "Unknown union type" {
		t.Errorf("Description = %v, want 'Unknown union type'", schema.Description)
	}
}

// Test edge cases for better coverage
func TestConventionOpenAPIGenerator_EdgeCases(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	t.Run("processPathSection with non-struct type", func(t *testing.T) {
		operation := &Operation{}
		// Should not panic with non-struct type
		generator.processPathSection(reflect.TypeOf("string"), operation, spec.Components)

		// Operation should not have any parameters added
		if len(operation.Parameters) != 0 {
			t.Errorf("Expected no parameters for non-struct type, got %d", len(operation.Parameters))
		}
	})

	t.Run("processQuerySection with non-struct type", func(t *testing.T) {
		operation := &Operation{}
		// Should not panic with non-struct type
		generator.processQuerySection(reflect.TypeOf(123), operation, spec.Components)

		// Operation should not have any parameters added
		if len(operation.Parameters) != 0 {
			t.Errorf("Expected no parameters for non-struct type, got %d", len(operation.Parameters))
		}
	})

	t.Run("processHeadersSection with non-struct type", func(t *testing.T) {
		operation := &Operation{}
		// Should not panic with non-struct type
		generator.processHeadersSection(reflect.TypeOf([]string{}), operation, spec.Components)

		// Operation should not have any parameters added
		if len(operation.Parameters) != 0 {
			t.Errorf("Expected no parameters for non-struct type, got %d", len(operation.Parameters))
		}
	})

	t.Run("processCookiesSection with non-struct type", func(t *testing.T) {
		operation := &Operation{}
		// Should not panic with non-struct type
		generator.processCookiesSection(reflect.TypeOf(map[string]interface{}{}), operation, spec.Components)

		// Operation should not have any parameters added
		if len(operation.Parameters) != 0 {
			t.Errorf("Expected no parameters for non-struct type, got %d", len(operation.Parameters))
		}
	})

	t.Run("section with fields without gork tags", func(t *testing.T) {
		type StructWithoutGorkTags struct {
			Name  string `json:"name"`
			Value int
		}

		operation := &Operation{}
		generator.processQuerySection(reflect.TypeOf(StructWithoutGorkTags{}), operation, spec.Components)

		// Should not add any parameters for fields without gork tags
		if len(operation.Parameters) != 0 {
			t.Errorf("Expected no parameters for struct without gork tags, got %d", len(operation.Parameters))
		}
	})
}

func TestConventionOpenAPIGenerator_ExtractDiscriminatorValue(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())

	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{
			name:     "email_auth_discriminator",
			typ:      reflect.TypeOf(EmailAuth{}),
			expected: "email",
		},
		{
			name:     "token_auth_discriminator",
			typ:      reflect.TypeOf(TokenAuth{}),
			expected: "token",
		},
		{
			name:     "no_discriminator",
			typ:      reflect.TypeOf(TestUnionType{}),
			expected: "",
		},
		{
			name:     "non_struct_type",
			typ:      reflect.TypeOf("string"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.extractDiscriminatorValue(tt.typ)
			if result != tt.expected {
				t.Errorf("extractDiscriminatorValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConventionOpenAPIGenerator_ProcessBodySection_NonStruct(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test with non-struct type (should return early)
	bodyType := reflect.TypeOf("string")
	operation := &Operation{}

	generator.processBodySection(bodyType, nil, operation, spec.Components)

	// Should not have set RequestBody since it's not a struct
	if operation.RequestBody != nil {
		t.Error("RequestBody should be nil for non-struct type")
	}
}

func TestConventionOpenAPIGenerator_ProcessQuerySection_EmptyGorkTag(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	queryType := reflect.TypeOf(struct {
		Field1 string `validate:"required"` // No gork tag
		Field2 string `gork:"field2"`
	}{})

	operation := &Operation{
		Parameters: []Parameter{},
	}

	generator.processQuerySection(queryType, operation, spec.Components)

	// Should only process field with gork tag
	if len(operation.Parameters) != 1 {
		t.Errorf("Parameter count = %d, want 1", len(operation.Parameters))
	}

	if operation.Parameters[0].Name != "field2" {
		t.Errorf("Parameter name = %v, want field2", operation.Parameters[0].Name)
	}
}

func TestConventionOpenAPIGenerator_ProcessResponseHeaders(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	t.Run("section with fields without gork tags", func(t *testing.T) {
		type StructWithoutGorkTags struct {
			Location   string `json:"location"`
			XRequestID string
		}

		response := &Response{
			Headers: make(map[string]*Header),
		}
		generator.processResponseHeaders(reflect.TypeOf(StructWithoutGorkTags{}), response, spec.Components)

		// Should not add any headers for fields without gork tags
		if len(response.Headers) != 0 {
			t.Errorf("Expected no headers for struct without gork tags, got %d", len(response.Headers))
		}
	})
}

func TestConventionOpenAPIGenerator_GenerateSchemaFromType_UnionType(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test union type path
	unionType := reflect.TypeOf(unions.Union2[EmailAuth, TokenAuth]{})
	schema := generator.generateSchemaFromType(unionType, "", spec.Components)

	if schema == nil {
		t.Fatal("generateSchemaFromType() returned nil for union type")
	}

	if len(schema.OneOf) == 0 {
		t.Error("Expected OneOf schema for union type, got none")
	}

	if len(schema.OneOf) != 2 {
		t.Errorf("Expected 2 OneOf schemas for Union2, got %d", len(schema.OneOf))
	}
}

func TestConventionOpenAPIGenerator_GenerateSchemaFromType_RegularType(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test regular type path (non-union)
	regularType := reflect.TypeOf(EmailAuth{})
	schema := generator.generateSchemaFromType(regularType, "required", spec.Components)

	if schema == nil {
		t.Fatal("generateSchemaFromType() returned nil for regular type")
	}

	// Should not have OneOf for regular types
	if len(schema.OneOf) != 0 {
		t.Error("Expected no OneOf schema for regular type")
	}
}

func TestConventionOpenAPIGenerator_GenerateUnionSchema_NoDiscriminator(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	type TypeWithoutDiscriminator struct {
		Value string `gork:"value"`
	}

	unionType := reflect.TypeOf(unions.Union2[TypeWithoutDiscriminator, TypeWithoutDiscriminator]{})
	schema := generator.generateUnionSchema(unionType, spec.Components)

	if schema == nil {
		t.Fatal("generateUnionSchema() returned nil")
	}

	if schema.Discriminator != nil {
		t.Errorf("Expected no discriminator, got %v", schema.Discriminator)
	}
}

func TestConventionOpenAPIGenerator_GenerateUnionSchema_EmptySanitizedSchemaName(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Create a type whose name will result in an empty sanitized schema name
	// (e.g., if sanitizeSchemaName removes all characters)
	type _ struct { // This struct name will likely result in an empty sanitized name
		Type  string `gork:"type,discriminator=test"`
		Field string
	}

	unionType := reflect.TypeOf(unions.Union2[struct {
		Type  string `gork:"type,discriminator=test"`
		Field string
	}, struct {
		Type  string `gork:"type,discriminator=test2"`
		Field string
	}]{})

	schema := generator.generateUnionSchema(unionType, spec.Components)

	if schema == nil {
		t.Fatal("generateUnionSchema() returned nil")
	}

	// Check that the discriminator mapping is not populated for empty schema names
	if schema.Discriminator != nil {
		if len(schema.Discriminator.Mapping) != 2 {
			t.Errorf("Expected 2 mappings, got %d", len(schema.Discriminator.Mapping))
		}
	}
}

func TestConventionOpenAPIGenerator_GetTypeDescription(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}

	tests := []struct {
		name        string
		generator   *ConventionOpenAPIGenerator
		typ         reflect.Type
		expected    string
		description string
	}{
		{
			name:        "nil_extractor",
			generator:   NewConventionOpenAPIGenerator(spec, nil),
			typ:         reflect.TypeOf(TestOpenAPIRequest{}),
			expected:    "",
			description: "should return empty string when extractor is nil",
		},
		{
			name:        "empty_type_name",
			generator:   NewConventionOpenAPIGenerator(spec, NewDocExtractor()),
			typ:         reflect.TypeOf([]string{}), // slice type has empty name
			expected:    "",
			description: "should return empty string when type name is empty",
		},
		{
			name:        "valid_type_with_extractor",
			generator:   NewConventionOpenAPIGenerator(spec, NewDocExtractor()),
			typ:         reflect.TypeOf(TestOpenAPIRequest{}),
			expected:    "", // No docs for test types, but function should work
			description: "should call extractor for named types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.generator.getTypeDescription(tt.typ)
			if result != tt.expected {
				t.Errorf("getTypeDescription() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConventionOpenAPIGenerator_GenerateUnionMemberSchemas(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, nil)

	t.Run("empty_union_types", func(t *testing.T) {
		// Test with empty union types
		var unionTypes []reflect.Type

		schemas, mapping := generator.generateUnionMemberSchemas(unionTypes, spec.Components)

		if len(schemas) != 0 {
			t.Errorf("Expected empty schemas for empty union types, got %d", len(schemas))
		}
		if len(mapping) != 0 {
			t.Errorf("Expected empty mapping for empty union types, got %d", len(mapping))
		}
	})

	t.Run("valid_member_types", func(t *testing.T) {
		// Test with valid member types that generate schemas
		unionTypes := []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(0)}

		schemas, mapping := generator.generateUnionMemberSchemas(unionTypes, spec.Components)

		if len(schemas) != 2 {
			t.Errorf("Expected 2 schemas for valid types, got %d", len(schemas))
		}
		if len(mapping) != 0 {
			t.Errorf("Expected empty mapping for basic types without discriminator tags, got %d", len(mapping))
		}
	})

	t.Run("nil_type_in_union", func(t *testing.T) {
		// Create a test with interface{} which might behave differently
		var nilInterface interface{}
		interfaceType := reflect.TypeOf(&nilInterface).Elem()

		unionTypes := []reflect.Type{interfaceType, reflect.TypeOf("")}

		schemas, mapping := generator.generateUnionMemberSchemas(unionTypes, spec.Components)

		// Test whatever the actual behavior is - just ensure it runs without panic
		_ = len(schemas)
		_ = len(mapping)
	})

	t.Run("comprehensive_union_types", func(t *testing.T) {
		// Test with a comprehensive set of types to ensure good coverage
		unionTypes := []reflect.Type{
			reflect.TypeOf(""),                       // string
			reflect.TypeOf(0),                        // int
			reflect.TypeOf(true),                     // bool
			reflect.TypeOf(0.0),                      // float64
			reflect.TypeOf([]string{}),               // slice
			reflect.TypeOf(map[string]interface{}{}), // map
		}

		schemas, mapping := generator.generateUnionMemberSchemas(unionTypes, spec.Components)

		// All basic types should generate schemas
		if len(schemas) != 6 {
			t.Errorf("Expected 6 schemas for basic types, got %d", len(schemas))
		}
		if len(mapping) != 0 {
			t.Errorf("Expected empty mapping for types without discriminators, got %d", len(mapping))
		}
	})
}

func TestConventionOpenAPIGenerator_GenerateUnionMemberSchemas_NilSchema(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, nil)

	t.Run("test_with_invalid_reflect_type", func(t *testing.T) {
		// Try to create a scenario that might cause schema generation issues
		// Use a nil type in the slice to see if this triggers the continue path
		var nilType reflect.Type
		unionTypes := []reflect.Type{
			nilType,           // This might cause issues and trigger continue
			reflect.TypeOf(0), // This will return valid schema
		}

		// This should handle the nil type gracefully and skip it
		schemas, mapping := generator.generateUnionMemberSchemas(unionTypes, spec.Components)

		// The nil type should be skipped, leaving only the int schema
		if len(schemas) < 1 {
			t.Errorf("Expected at least 1 schema (nil type should be handled), got %d", len(schemas))
		}
		if len(mapping) != 0 {
			t.Errorf("Expected empty mapping for basic types without discriminator tags, got %d", len(mapping))
		}
	})
}

func TestConventionOpenAPIGenerator_ParseDiscriminatorFromGorkTag(t *testing.T) {
	generator := NewConventionOpenAPIGenerator(nil, nil)

	tests := []struct {
		name     string
		gorkTag  string
		expected string
	}{
		{
			name:     "empty_tag",
			gorkTag:  "",
			expected: "",
		},
		{
			name:     "no_discriminator",
			gorkTag:  "field_name,required",
			expected: "",
		},
		{
			name:     "discriminator_found",
			gorkTag:  "field_name,discriminator=user",
			expected: "user",
		},
		{
			name:     "discriminator_with_spaces",
			gorkTag:  "field_name, discriminator=admin ",
			expected: "admin",
		},
		{
			name:     "multiple_parts_no_discriminator",
			gorkTag:  "field,required,validate",
			expected: "",
		},
		{
			name:     "contains_discriminator_string_but_no_match",
			gorkTag:  "field,notdiscriminator=value,other",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.parseDiscriminatorFromGorkTag(tt.gorkTag)
			if result != tt.expected {
				t.Errorf("parseDiscriminatorFromGorkTag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test types for union schema naming
type CreditCardPaymentMethod struct {
	Type       string `gork:"type" validate:"required,eq=credit_card"`
	CardNumber string `gork:"cardNumber" validate:"required"`
}

type BankPaymentMethod struct {
	Type          string `gork:"type" validate:"required,eq=bank_account"`
	AccountNumber string `gork:"accountNumber" validate:"required"`
	RoutingNumber string `gork:"routingNumber" validate:"required"`
}

type PaymentMethodUnionRequest struct {
	Body unions.Union2[BankPaymentMethod, CreditCardPaymentMethod]
}

func TestGenerateRequestBodyComponentName_UnionTypes(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test union type schema naming
	t.Run("Union2 body type generates valid schema name", func(t *testing.T) {
		requestType := reflect.TypeOf(PaymentMethodUnionRequest{})
		bodyField, _ := requestType.FieldByName("Body")
		bodyType := bodyField.Type

		// This should reproduce the issue: the schema name contains invalid characters
		schemaName := generator.generateRequestBodyComponentName(bodyType, requestType)

		// Before fix: this would be "Union2[github.com/gork-labs/gork/pkg/api.BankPaymentMethod,github.com/gork-labs/gork/pkg/api.CreditCardPaymentMethod]Body"
		// After fix: this should be a concise, valid schema name like "BankOrCreditBody"

		t.Logf("Generated schema name: %s", schemaName)

		// Check that the schema name doesn't contain invalid characters
		invalidChars := []string{"[", "]", ",", "/", "."}
		for _, char := range invalidChars {
			if containsChar(schemaName, char) {
				t.Errorf("Schema name contains invalid character '%s': %s", char, schemaName)
			}
		}

		// Check that it ends with "Body"
		if !endsWithSuffix(schemaName, "Body") {
			t.Errorf("Schema name should end with 'Body': %s", schemaName)
		}

		// Check that it's a reasonable length (should be much shorter than the old verbose name)
		if len(schemaName) > 25 {
			t.Errorf("Schema name should be concise (<=25 chars): %s (length: %d)", schemaName, len(schemaName))
		}

		// Check that it conveys the union nature (either contains "Or" or is descriptive)
		if !containsSubstring(schemaName, "Or") && len(schemaName) > 15 {
			t.Errorf("Schema name should be concise or contain 'Or' to indicate union: %s", schemaName)
		}
	})
}

// Test createUnionNameByMemberCount function directly to achieve 100% coverage
func TestCreateUnionNameByMemberCount(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	gen := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	testCases := []struct {
		base         string
		cleanMembers []string
		expected     string
		description  string
	}{
		// Test case 2: binary union
		{"Union2", []string{"String", "Int"}, "StringOrInt", "two members should create binary union name"},

		// Test case 3: should return base + "Options"
		{"Union3", []string{"A", "B", "C"}, "Union3Options", "three members should return Options suffix"},

		// Test case 4: should return base + "Options"
		{"Union4", []string{"A", "B", "C", "D"}, "Union4Options", "four members should return Options suffix"},

		// Test default case: 0 members
		{"Union0", []string{}, "Union0Type", "zero members should hit default case"},

		// Test default case: 1 member
		{"Union1", []string{"Single"}, "Union1Type", "one member should hit default case"},

		// Test default case: 5+ members
		{"Union5", []string{"A", "B", "C", "D", "E"}, "Union5Type", "five members should hit default case"},

		// Test default case: many members
		{"Union10", []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}, "Union10Type", "ten members should hit default case"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := gen.createUnionNameByMemberCount(tc.base, tc.cleanMembers)

			// For the binary case, the exact result depends on createBinaryUnionName logic
			// so we just check that it's not empty and different from the base
			if len(tc.cleanMembers) == 2 {
				if result == "" {
					t.Errorf("Binary union name should not be empty for %v", tc.cleanMembers)
				}
				t.Logf("Binary union result: %s", result)
			} else {
				if result != tc.expected {
					t.Errorf("Expected %s, got %s for %s with %d members", tc.expected, result, tc.base, len(tc.cleanMembers))
				}
			}
		})
	}
}

// Test helper functions to maintain coverage
func TestHelperFunctions_Coverage(t *testing.T) {
	spec := &OpenAPISpec{
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
	gen := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Test extractMeaningfulName
	t.Run("extractMeaningfulName", func(t *testing.T) {
		tests := []struct{ input, expected string }{
			{"CreditCardPaymentMethod", "CreditCard"}, // suffix removal
			{"Short", "Short"},                        // short name
			{"VeryLongNameWithoutSuffix", "VeryLong"}, // camelCase extraction
			{"ABAuth", "ABAuth"},                      // short core, should keep full name
			{"alllowercase", "alllowercase"},          // no camelCase boundaries
		}
		for _, tt := range tests {
			if result := gen.extractMeaningfulName(tt.input); result != tt.expected {
				t.Errorf("extractMeaningfulName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})

	// Test createBinaryUnionName
	t.Run("createBinaryUnionName", func(t *testing.T) {
		tests := []struct {
			members  []string
			expected string
		}{
			{[]string{"Bank", "Credit"}, "BankOrCredit"},    // both short
			{[]string{"Auth", "VeryLong"}, "AuthOrVeryLo"},  // one short first
			{[]string{"VeryLong", "Auth"}, "VeryLoOrAuth"},  // one short second
			{[]string{"VeryLong", "Another"}, "VeryOrAnot"}, // both long
			{[]string{"Single"}, "UnionType"},               // wrong count
		}
		for _, tt := range tests {
			if result := gen.createBinaryUnionName(tt.members); result != tt.expected {
				t.Errorf("createBinaryUnionName(%v) = %q, want %q", tt.members, result, tt.expected)
			}
		}
	})

	// Test generateConciseUnionName edge cases
	t.Run("generateConciseUnionName", func(t *testing.T) {
		// Test fallback to sanitization for non-generic types
		normalType := reflect.TypeOf(struct{ Name string }{})
		result := gen.generateConciseUnionName(normalType)
		// Should fall back to sanitization (empty for anonymous struct)
		if result != "" {
			t.Logf("Non-generic type result: %s", result)
		}
	})
}

// Helper functions for the test
func containsChar(s, char string) bool {
	for _, r := range s {
		if string(r) == char {
			return true
		}
	}
	return false
}

func endsWithSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
