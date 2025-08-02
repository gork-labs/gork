// Code generated test file for adapter
package api

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"
)

type queryTestReq struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Count   uint     `json:"count"`
	Active  bool     `json:"active"`
	Tags    []string `json:"tags"`
	CSVTags []string `json:"csvTags"`
}

func TestParseQueryParams(t *testing.T) {
	reqStruct := &queryTestReq{}

	r := httptest.NewRequest("GET", "/test?name=Alice&age=30&count=5&active=true&tags=foo&tags=bar&csvTags=alpha,beta", nil)

	parseQueryParams(r, reqStruct)

	if reqStruct.Name != "Alice" {
		t.Errorf("expected Name to be 'Alice', got %q", reqStruct.Name)
	}
	if reqStruct.Age != 30 {
		t.Errorf("expected Age to be 30, got %d", reqStruct.Age)
	}
	if reqStruct.Count != 5 {
		t.Errorf("expected Count to be 5, got %d", reqStruct.Count)
	}
	if !reqStruct.Active {
		t.Errorf("expected Active to be true")
	}
	expectedTags := []string{"foo", "bar"}
	if len(reqStruct.Tags) != len(expectedTags) {
		t.Fatalf("expected %d tags, got %d", len(expectedTags), len(reqStruct.Tags))
	}
	for i, tag := range expectedTags {
		if reqStruct.Tags[i] != tag {
			t.Errorf("expected Tags[%d] to be %q, got %q", i, tag, reqStruct.Tags[i])
		}
	}

	// Verify CSV-style slice parsing with a single additional check
	if len(reqStruct.CSVTags) != 2 || reqStruct.CSVTags[0] != "alpha" || reqStruct.CSVTags[1] != "beta" {
		t.Errorf("expected CSVTags to be [alpha beta], got %v", reqStruct.CSVTags)
	}
}

// DummyHandler is used to test getFunctionName
func DummyHandler(ctx context.Context, req queryTestReq) (string, error) {
	return "", nil
}

func TestGetFunctionName(t *testing.T) {
	name := getFunctionName(DummyHandler)
	if name != "DummyHandler" {
		t.Errorf("expected function name 'DummyHandler', got %q", name)
	}
}

// Tests for option functions
func TestWithTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{
			name: "single tag",
			tags: []string{"api"},
			want: []string{"api"},
		},
		{
			name: "multiple tags",
			tags: []string{"api", "users", "v1"},
			want: []string{"api", "users", "v1"},
		},
		{
			name: "empty tags",
			tags: []string{},
			want: []string{},
		},
		{
			name: "duplicate tags",
			tags: []string{"api", "api", "users"},
			want: []string{"api", "api", "users"}, // WithTags doesn't deduplicate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := WithTags(tt.tags...)
			handlerOption := &HandlerOption{}

			option(handlerOption)

			if len(handlerOption.Tags) != len(tt.want) {
				t.Errorf("WithTags() tags length = %d, want %d", len(handlerOption.Tags), len(tt.want))
			}

			for i, tag := range tt.want {
				if handlerOption.Tags[i] != tag {
					t.Errorf("WithTags() tags[%d] = %s, want %s", i, handlerOption.Tags[i], tag)
				}
			}
		})
	}
}

func TestWithTags_Append(t *testing.T) {
	// Test that WithTags appends to existing tags
	handlerOption := &HandlerOption{
		Tags: []string{"existing"},
	}

	option := WithTags("new1", "new2")
	option(handlerOption)

	expected := []string{"existing", "new1", "new2"}
	if len(handlerOption.Tags) != len(expected) {
		t.Errorf("WithTags() append length = %d, want %d", len(handlerOption.Tags), len(expected))
	}

	for i, tag := range expected {
		if handlerOption.Tags[i] != tag {
			t.Errorf("WithTags() append tags[%d] = %s, want %s", i, handlerOption.Tags[i], tag)
		}
	}
}

func TestWithBasicAuth(t *testing.T) {
	option := WithBasicAuth()
	handlerOption := &HandlerOption{}

	option(handlerOption)

	if len(handlerOption.Security) != 1 {
		t.Errorf("WithBasicAuth() security length = %d, want 1", len(handlerOption.Security))
	}

	security := handlerOption.Security[0]
	if security.Type != "basic" {
		t.Errorf("WithBasicAuth() security type = %s, want basic", security.Type)
	}

	if len(security.Scopes) != 0 {
		t.Errorf("WithBasicAuth() security scopes = %v, want empty", security.Scopes)
	}
}

func TestWithBearerTokenAuth(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		want   []string
	}{
		{
			name:   "no scopes",
			scopes: []string{},
			want:   []string{},
		},
		{
			name:   "single scope",
			scopes: []string{"read"},
			want:   []string{"read"},
		},
		{
			name:   "multiple scopes",
			scopes: []string{"read", "write", "admin"},
			want:   []string{"read", "write", "admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := WithBearerTokenAuth(tt.scopes...)
			handlerOption := &HandlerOption{}

			option(handlerOption)

			if len(handlerOption.Security) != 1 {
				t.Errorf("WithBearerTokenAuth() security length = %d, want 1", len(handlerOption.Security))
			}

			security := handlerOption.Security[0]
			if security.Type != "bearer" {
				t.Errorf("WithBearerTokenAuth() security type = %s, want bearer", security.Type)
			}

			if len(security.Scopes) != len(tt.want) {
				t.Errorf("WithBearerTokenAuth() scopes length = %d, want %d", len(security.Scopes), len(tt.want))
			}

			for i, scope := range tt.want {
				if security.Scopes[i] != scope {
					t.Errorf("WithBearerTokenAuth() scopes[%d] = %s, want %s", i, security.Scopes[i], scope)
				}
			}
		})
	}
}

func TestWithAPIKeyAuth(t *testing.T) {
	option := WithAPIKeyAuth()
	handlerOption := &HandlerOption{}

	option(handlerOption)

	if len(handlerOption.Security) != 1 {
		t.Errorf("WithAPIKeyAuth() security length = %d, want 1", len(handlerOption.Security))
	}

	security := handlerOption.Security[0]
	if security.Type != "apiKey" {
		t.Errorf("WithAPIKeyAuth() security type = %s, want apiKey", security.Type)
	}

	if len(security.Scopes) != 0 {
		t.Errorf("WithAPIKeyAuth() security scopes = %v, want empty", security.Scopes)
	}
}

func TestSecurityRequirement_MultipleOptions(t *testing.T) {
	// Test combining multiple security options
	handlerOption := &HandlerOption{}

	// Apply multiple security options
	WithBasicAuth()(handlerOption)
	WithBearerTokenAuth("read", "write")(handlerOption)
	WithAPIKeyAuth()(handlerOption)

	if len(handlerOption.Security) != 3 {
		t.Errorf("Multiple security options length = %d, want 3", len(handlerOption.Security))
	}

	// Verify basic auth
	if handlerOption.Security[0].Type != "basic" {
		t.Errorf("Security[0] type = %s, want basic", handlerOption.Security[0].Type)
	}

	// Verify bearer auth
	if handlerOption.Security[1].Type != "bearer" {
		t.Errorf("Security[1] type = %s, want bearer", handlerOption.Security[1].Type)
	}
	if len(handlerOption.Security[1].Scopes) != 2 {
		t.Errorf("Security[1] scopes length = %d, want 2", len(handlerOption.Security[1].Scopes))
	}

	// Verify API key auth
	if handlerOption.Security[2].Type != "apiKey" {
		t.Errorf("Security[2] type = %s, want apiKey", handlerOption.Security[2].Type)
	}
}

func TestHandlerOption_CombinedOptions(t *testing.T) {
	// Test combining tags and security options
	handlerOption := &HandlerOption{}

	WithTags("api", "v1")(handlerOption)
	WithBasicAuth()(handlerOption)
	WithTags("auth")(handlerOption) // Should append to existing tags

	expectedTags := []string{"api", "v1", "auth"}
	if len(handlerOption.Tags) != len(expectedTags) {
		t.Errorf("Combined options tags length = %d, want %d", len(handlerOption.Tags), len(expectedTags))
	}

	for i, tag := range expectedTags {
		if handlerOption.Tags[i] != tag {
			t.Errorf("Combined options tags[%d] = %s, want %s", i, handlerOption.Tags[i], tag)
		}
	}

	if len(handlerOption.Security) != 1 {
		t.Errorf("Combined options security length = %d, want 1", len(handlerOption.Security))
	}
}

func TestHandlerOption_EmptyInitialization(t *testing.T) {
	// Test that HandlerOption initializes with empty slices
	handlerOption := &HandlerOption{}

	if handlerOption.Tags != nil {
		t.Errorf("HandlerOption.Tags should be nil initially, got %v", handlerOption.Tags)
	}

	if handlerOption.Security != nil {
		t.Errorf("HandlerOption.Security should be nil initially, got %v", handlerOption.Security)
	}
}

// Benchmark tests for option functions
func BenchmarkWithTags(b *testing.B) {
	handlerOption := &HandlerOption{}
	option := WithTags("api", "users", "v1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		option(handlerOption)
	}
}

func BenchmarkWithBasicAuth(b *testing.B) {
	handlerOption := &HandlerOption{}
	option := WithBasicAuth()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		option(handlerOption)
	}
}

func BenchmarkWithBearerTokenAuth(b *testing.B) {
	handlerOption := &HandlerOption{}
	option := WithBearerTokenAuth("read", "write", "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		option(handlerOption)
	}
}

func BenchmarkWithAPIKeyAuth(b *testing.B) {
	handlerOption := &HandlerOption{}
	option := WithAPIKeyAuth()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		option(handlerOption)
	}
}

// Additional tests for improving coverage

func TestParseOpenAPITag_EmptyTag(t *testing.T) {
	result := parseOpenAPITag("")
	if result.Name != "" || result.In != "" {
		t.Errorf("Expected empty result for empty tag, got Name='%s', In='%s'", result.Name, result.In)
	}
}

func TestParseOpenAPITag_EmptyPart(t *testing.T) {
	result := parseOpenAPITag("field,,in=query")
	if result.Name != "field" || result.In != "query" {
		t.Errorf("Expected Name='field', In='query', got Name='%s', In='%s'", result.Name, result.In)
	}
}

func TestParseQueryParams_OpenAPITag(t *testing.T) {
	type TestReq struct {
		APIKey string `openapi:"api_key,in=query" json:"key"`
	}
	
	req := &TestReq{}
	r := httptest.NewRequest("GET", "/test?api_key=secret", nil)
	
	parseQueryParams(r, req)
	
	if req.APIKey != "secret" {
		t.Errorf("Expected 'secret', got '%s'", req.APIKey)
	}
}

func TestParseQueryParams_OpenAPITagNotQuery(t *testing.T) {
	type TestReq struct {
		APIKey string `openapi:"api_key,in=header" json:"key"`
	}
	
	req := &TestReq{}
	r := httptest.NewRequest("GET", "/test?api_key=secret", nil)
	
	parseQueryParams(r, req)
	
	// Should not set value for non-query openapi tags
	if req.APIKey != "" {
		t.Errorf("Expected empty string, got '%s'", req.APIKey)
	}
}

func TestSetFieldValue_SliceType(t *testing.T) {
	type TestStruct struct {
		Tags []string
	}
	
	s := &TestStruct{}
	v := reflect.ValueOf(s).Elem()
	field := reflect.TypeOf(s).Elem().Field(0)
	
	setFieldValue(v.Field(0), field, "a,b,c", []string{"a,b,c"})
	
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(s.Tags, expected) {
		t.Errorf("Expected %v, got %v", expected, s.Tags)
	}
}

func TestSetSliceFieldValue_SingleValueWithComma(t *testing.T) {
	type TestStruct struct {
		Tags []string
	}
	
	s := &TestStruct{}
	v := reflect.ValueOf(s).Elem()
	field := reflect.TypeOf(s).Elem().Field(0)
	
	setSliceFieldValue(v.Field(0), field, "a,b,c", []string{"a,b,c"})
	
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(s.Tags, expected) {
		t.Errorf("Expected %v, got %v", expected, s.Tags)
	}
}

func TestSetSliceFieldValue_EmptyParam(t *testing.T) {
	type TestStruct struct {
		Tags []string
	}
	
	s := &TestStruct{}
	v := reflect.ValueOf(s).Elem()
	field := reflect.TypeOf(s).Elem().Field(0)
	
	setSliceFieldValue(v.Field(0), field, "", []string{})
	
	// Should not set slice for empty values
	if s.Tags != nil {
		t.Errorf("Expected nil, got %v", s.Tags)
	}
}

func TestGetFunctionName_NilPointer(t *testing.T) {
	var nilFunc func()
	name := getFunctionName(nilFunc)
	
	// Should return empty string for nil function
	if name != "" {
		t.Errorf("Expected empty string, got '%s'", name)
	}
}

func TestGetFunctionName_WithPackagePath(t *testing.T) {
	// Test with a function that has a package path
	name := getFunctionName(DummyHandler)
	
	// Should strip package path and return just the function name
	if name != "DummyHandler" {
		t.Errorf("Expected 'DummyHandler', got '%s'", name)
	}
}
