package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// Test types for comprehensive parameter testing
type queryTestReq struct {
	Name      string   `json:"name"`
	Age       int      `json:"age"`
	Count     uint     `json:"count"`
	Active    bool     `json:"active"`
	Tags      []string `json:"tags"`
	CSVTags   []string `json:"csvTags"`
	Rating    float64  `json:"rating"`
	APIKey    string   `openapi:"api_key,in=query" json:"apiKey"`
	HeaderVal string   `openapi:"x-custom,in=header" json:"headerVal"`
	CookieVal string   `openapi:"session,in=cookie" json:"cookieVal"`
	PathID    string   `openapi:"id,in=path" json:"pathId"`
}

type testReqWithOpenAPI struct {
	UserID   string `openapi:"user_id,in=path" json:"userId"`
	Limit    int    `openapi:"limit,in=query" json:"limit"`
	Token    string `openapi:"auth-token,in=header" json:"token"`
	Session  string `openapi:"session_id,in=cookie" json:"session"`
	BodyData string `json:"bodyData"`
}

// TestParameterParsing consolidates all parameter parsing tests
func TestParameterParsing(t *testing.T) {
	t.Run("query parameters", func(t *testing.T) {
		tests := []struct {
			name     string
			url      string
			expected queryTestReq
		}{
			{
				name: "basic query params",
				url:  "/test?name=Alice&age=30&count=5&active=true&rating=4.5",
				expected: queryTestReq{
					Name:   "Alice",
					Age:    30,
					Count:  5,
					Active: true,
					Rating: 4.5,
				},
			},
			{
				name: "slice parameters",
				url:  "/test?tags=foo&tags=bar&csvTags=alpha,beta,gamma",
				expected: queryTestReq{
					Tags:    []string{"foo", "bar"},
					CSVTags: []string{"alpha", "beta", "gamma"},
				},
			},
			{
				name: "openapi query param",
				url:  "/test?api_key=secret123",
				expected: queryTestReq{
					APIKey: "secret123",
				},
			},
			{
				name: "empty values",
				url:  "/test?name=&age=0&active=false",
				expected: queryTestReq{
					Name:   "",
					Age:    0,
					Active: false,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := &queryTestReq{}
				r := httptest.NewRequest("GET", tt.url, nil)

				parseQueryParams(r, req)

				if req.Name != tt.expected.Name {
					t.Errorf("Name: got %q, want %q", req.Name, tt.expected.Name)
				}
				if req.Age != tt.expected.Age {
					t.Errorf("Age: got %d, want %d", req.Age, tt.expected.Age)
				}
				if req.Count != tt.expected.Count {
					t.Errorf("Count: got %d, want %d", req.Count, tt.expected.Count)
				}
				if req.Active != tt.expected.Active {
					t.Errorf("Active: got %t, want %t", req.Active, tt.expected.Active)
				}
				if req.Rating != tt.expected.Rating {
					t.Errorf("Rating: got %f, want %f", req.Rating, tt.expected.Rating)
				}
				if req.APIKey != tt.expected.APIKey {
					t.Errorf("APIKey: got %q, want %q", req.APIKey, tt.expected.APIKey)
				}
				if !reflect.DeepEqual(req.Tags, tt.expected.Tags) {
					t.Errorf("Tags: got %v, want %v", req.Tags, tt.expected.Tags)
				}
				if !reflect.DeepEqual(req.CSVTags, tt.expected.CSVTags) {
					t.Errorf("CSVTags: got %v, want %v", req.CSVTags, tt.expected.CSVTags)
				}
			})
		}
	})

	t.Run("malformed values", func(t *testing.T) {
		req := &queryTestReq{}
		r := httptest.NewRequest("GET", "/test?age=invalid&active=maybe&rating=notanumber", nil)

		parseQueryParams(r, req)

		// Should handle malformed values gracefully
		if req.Age != 0 {
			t.Errorf("Age should be 0 for invalid input, got %d", req.Age)
		}
		if req.Active != false {
			t.Errorf("Active should be false for invalid input, got %t", req.Active)
		}
		// For float parsing, "notanumber" should result in 0, not NaN
		if req.Rating != 0 {
			t.Errorf("Rating should be 0 for invalid input, got %f", req.Rating)
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test", nil)

		// Test with nil request - this should panic (expected behavior)
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("parseQueryParams with nil should panic, but didn't")
				}
			}()
			var nilReq *queryTestReq
			parseQueryParams(r, nilReq)
		}()

		// Test with non-struct type - this should also panic (expected behavior)
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("parseQueryParams with non-struct should panic, but didn't")
				}
			}()
			s := "not a struct"
			parseQueryParams(r, &s)
		}()
	})
}

// TestFunctionNameExtraction consolidates all function name tests
func TestFunctionNameExtraction(t *testing.T) {
	// Test handlers for function name extraction
	handler1 := func(ctx context.Context, req queryTestReq) (string, error) { return "", nil }
	handler2 := func(ctx context.Context, req testReqWithOpenAPI) (*testReqWithOpenAPI, error) { return nil, nil }

	tests := []struct {
		name     string
		handler  interface{}
		expected string
	}{
		{
			name:     "named function",
			handler:  DummyHandler,
			expected: "DummyHandler",
		},
		{
			name:     "anonymous function 1",
			handler:  handler1,
			expected: "TestFunctionNameExtraction.func1", // Go generates this name
		},
		{
			name:     "anonymous function 2",
			handler:  handler2,
			expected: "TestFunctionNameExtraction.func2",
		},
		{
			name:     "nil handler",
			handler:  nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil handler" {
				// Test that nil handler panics (expected behavior)
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("getFunctionName with nil should panic, but didn't")
					}
				}()
				getFunctionName(tt.handler)
			} else {
				name := getFunctionName(tt.handler)
				if tt.name == "named function" {
					if name != tt.expected {
						t.Errorf("expected %q, got %q", tt.expected, name)
					}
				} else {
					// For anonymous functions, just check it's not empty
					if name == "" {
						t.Errorf("expected non-empty name for anonymous function, got empty string")
					}
				}
			}
		})
	}

	t.Run("function name trimming", func(t *testing.T) {
		// Test trimming of package paths and prefixes
		tests := []struct {
			input    string
			expected string
		}{
			{"github.com/gork-labs/gork/pkg/api.Handler", "Handler"},
			{"main.(*Server).HandleRequest-fm", "HandleRequest-fm"},
			{"SimpleHandler", "SimpleHandler"},
			{"", ""},
		}

		for _, tt := range tests {
			result := trimFunctionName(tt.input)
			if result != tt.expected {
				t.Errorf("trimFunctionName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})
}

// DummyHandler is used to test getFunctionName
func DummyHandler(ctx context.Context, req queryTestReq) (string, error) {
	return "", nil
}

// TestOpenAPITagParsing consolidates OpenAPI tag parsing tests
func TestOpenAPITagParsing(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected struct {
			Name string
			In   string
		}
	}{
		{
			name: "empty tag",
			tag:  "",
			expected: struct {
				Name string
				In   string
			}{"", ""},
		},
		{
			name: "basic query param",
			tag:  "api_key,in=query",
			expected: struct {
				Name string
				In   string
			}{"api_key", "query"},
		},
		{
			name: "header param",
			tag:  "auth-token,in=header",
			expected: struct {
				Name string
				In   string
			}{"auth-token", "header"},
		},
		{
			name: "path param",
			tag:  "user_id,in=path",
			expected: struct {
				Name string
				In   string
			}{"user_id", "path"},
		},
		{
			name: "cookie param",
			tag:  "session_id,in=cookie",
			expected: struct {
				Name string
				In   string
			}{"session_id", "cookie"},
		},
		{
			name: "empty part in middle",
			tag:  "field,,in=query",
			expected: struct {
				Name string
				In   string
			}{"field", "query"},
		},
		{
			name: "malformed tag",
			tag:  "just_name",
			expected: struct {
				Name string
				In   string
			}{"just_name", ""},
		},
		{
			name: "no name",
			tag:  ",in=query",
			expected: struct {
				Name string
				In   string
			}{"", "query"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOpenAPITag(tt.tag)
			if result.Name != tt.expected.Name {
				t.Errorf("Name: got %q, want %q", result.Name, tt.expected.Name)
			}
			if result.In != tt.expected.In {
				t.Errorf("In: got %q, want %q", result.In, tt.expected.In)
			}
		})
	}
}

// TestFieldValueSetting consolidates all field value setting tests
func TestFieldValueSetting(t *testing.T) {
	t.Run("basic types", func(t *testing.T) {
		type TestStruct struct {
			StringVal string
			IntVal    int
			UintVal   uint
			BoolVal   bool
			FloatVal  float64
		}

		tests := []struct {
			name      string
			fieldName string
			value     string
			expected  interface{}
		}{
			{"string", "StringVal", "hello", "hello"},
			{"int", "IntVal", "42", 42},
			{"uint", "UintVal", "123", uint(123)},
			{"bool true", "BoolVal", "true", true},
			{"bool false", "BoolVal", "false", false},
			{"float", "FloatVal", "3.14", 3.14},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := &TestStruct{}
				v := reflect.ValueOf(s).Elem()
				field, _ := reflect.TypeOf(s).Elem().FieldByName(tt.fieldName)
				fieldValue := v.FieldByName(tt.fieldName)

				setFieldValue(fieldValue, field, tt.value, []string{tt.value})

				actual := fieldValue.Interface()
				if actual != tt.expected {
					t.Errorf("setFieldValue(%s, %s): got %v, want %v", tt.fieldName, tt.value, actual, tt.expected)
				}
			})
		}
	})

	t.Run("slice types", func(t *testing.T) {
		type TestStruct struct {
			Tags     []string
			Numbers  []int
			Booleans []bool
		}

		tests := []struct {
			name      string
			fieldName string
			paramVal  string
			allVals   []string
			expected  interface{}
		}{
			{
				name:      "string slice from comma-separated",
				fieldName: "Tags",
				paramVal:  "a,b,c",
				allVals:   []string{"a,b,c"},
				expected:  []string{"a", "b", "c"},
			},
			{
				name:      "string slice from multiple values",
				fieldName: "Tags",
				paramVal:  "x",
				allVals:   []string{"x", "y", "z"},
				expected:  []string{"x", "y", "z"},
			},
			{
				name:      "empty slice",
				fieldName: "Tags",
				paramVal:  "",
				allVals:   []string{},
				expected:  []string(nil),
			},
			{
				name:      "int slice (not supported)",
				fieldName: "Numbers",
				paramVal:  "1,2,3",
				allVals:   []string{"1,2,3"},
				expected:  []int(nil), // Not supported, should remain nil
			},
			{
				name:      "bool slice (not supported)",
				fieldName: "Booleans",
				paramVal:  "true,false,true",
				allVals:   []string{"true,false,true"},
				expected:  []bool(nil), // Not supported, should remain nil
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := &TestStruct{}
				v := reflect.ValueOf(s).Elem()
				field, _ := reflect.TypeOf(s).Elem().FieldByName(tt.fieldName)
				fieldValue := v.FieldByName(tt.fieldName)

				setSliceFieldValue(fieldValue, field, tt.paramVal, tt.allVals)

				actual := fieldValue.Interface()
				if !reflect.DeepEqual(actual, tt.expected) {
					t.Errorf("setSliceFieldValue(%s): got %v, want %v", tt.fieldName, actual, tt.expected)
				}
			})
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		type TestStruct struct {
			IntVal   int
			FloatVal float64
			BoolVal  bool
		}

		tests := []struct {
			name      string
			fieldName string
			value     string
		}{
			{"invalid int", "IntVal", "not-a-number"},
			{"invalid float", "FloatVal", "not-a-float"},
			{"invalid bool", "BoolVal", "maybe"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := &TestStruct{}
				v := reflect.ValueOf(s).Elem()
				field, _ := reflect.TypeOf(s).Elem().FieldByName(tt.fieldName)
				fieldValue := v.FieldByName(tt.fieldName)

				// Should not panic and should leave field at zero value
				setFieldValue(fieldValue, field, tt.value, []string{tt.value})

				// Verify field remains at zero value
				actual := fieldValue.Interface()
				zeroVal := reflect.Zero(fieldValue.Type()).Interface()
				if actual != zeroVal {
					t.Errorf("setFieldValue with invalid %s: got %v, want %v (zero value)", tt.name, actual, zeroVal)
				}
			})
		}
	})

	t.Run("time types (not supported)", func(t *testing.T) {
		type TestStruct struct {
			TimeVal     time.Time
			DurationVal time.Duration
		}

		s := &TestStruct{}
		v := reflect.ValueOf(s).Elem()

		// Test time.Time - should remain zero value (not supported)
		timeField, _ := reflect.TypeOf(s).Elem().FieldByName("TimeVal")
		timeFieldValue := v.FieldByName("TimeVal")
		setFieldValue(timeFieldValue, timeField, "2023-01-01T12:00:00Z", []string{"2023-01-01T12:00:00Z"})

		zeroTime := time.Time{}
		if !s.TimeVal.Equal(zeroTime) {
			t.Errorf("Time field should remain zero (not supported): got %v, want %v", s.TimeVal, zeroTime)
		}

		// Test time.Duration - should remain zero value (not supported)
		durField, _ := reflect.TypeOf(s).Elem().FieldByName("DurationVal")
		durFieldValue := v.FieldByName("DurationVal")
		setFieldValue(durFieldValue, durField, "5m30s", []string{"5m30s"})

		if s.DurationVal != 0 {
			t.Errorf("Duration field should remain zero (not supported): got %v, want 0", s.DurationVal)
		}
	})
}

// TestHandlerOptions consolidates all handler option tests
func TestHandlerOptions(t *testing.T) {
	t.Run("WithTags", func(t *testing.T) {
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
	})

	t.Run("WithTags append", func(t *testing.T) {
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
	})

	t.Run("WithBasicAuth", func(t *testing.T) {
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
	})

	t.Run("WithBearerTokenAuth", func(t *testing.T) {
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
	})

	t.Run("WithAPIKeyAuth", func(t *testing.T) {
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
	})

	t.Run("multiple security options", func(t *testing.T) {
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
	})

	t.Run("combined options", func(t *testing.T) {
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
	})

	t.Run("empty initialization", func(t *testing.T) {
		// Test that HandlerOption initializes with empty slices
		handlerOption := &HandlerOption{}

		if handlerOption.Tags != nil {
			t.Errorf("HandlerOption.Tags should be nil initially, got %v", handlerOption.Tags)
		}

		if handlerOption.Security != nil {
			t.Errorf("HandlerOption.Security should be nil initially, got %v", handlerOption.Security)
		}
	})
}

// TestExtractFunctionNameFromRuntimeWithFunc tests the function name extraction with custom provider
func TestExtractFunctionNameFromRuntimeWithFunc(t *testing.T) {
	t.Run("with nil function provider result", func(t *testing.T) {
		// Mock provider that returns nil (simulates FuncForPC failing)
		mockProvider := func(uintptr) *runtime.Func {
			return nil
		}

		result := extractFunctionNameFromRuntimeWithFunc(DummyHandler, mockProvider)
		if result != "" {
			t.Errorf("Expected empty string for nil function, got %q", result)
		}
	})

	t.Run("with custom function provider", func(t *testing.T) {
		// Test with the real provider to verify the behavior
		result := extractFunctionNameFromRuntimeWithFunc(DummyHandler, runtime.FuncForPC)
		if result == "" {
			t.Error("Expected non-empty result with real function")
		}
		if result != "DummyHandler" {
			t.Errorf("Expected 'DummyHandler', got %q", result)
		}
	})
}

// TestHTTPParameterAdapter consolidates HTTP parameter adapter tests
func TestHTTPParameterAdapter(t *testing.T) {
	adapter := HTTPParameterAdapter{}

	t.Run("Query", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test?name=Alice&age=30&empty=", nil)

		// Test existing parameter
		value, exists := adapter.Query(r, "name")
		if !exists {
			t.Error("Query parameter 'name' should exist")
		}
		if value != "Alice" {
			t.Errorf("Query parameter 'name': got %q, want %q", value, "Alice")
		}

		// Test non-existing parameter
		value, exists = adapter.Query(r, "nonexistent")
		if exists {
			t.Error("Query parameter 'nonexistent' should not exist")
		}
		if value != "" {
			t.Errorf("Non-existing query parameter: got %q, want empty string", value)
		}

		// Test empty parameter
		value, exists = adapter.Query(r, "empty")
		if exists {
			t.Error("Empty query parameter should not exist")
		}
	})

	t.Run("Header", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test", nil)
		r.Header.Set("X-Custom-Header", "custom-value")
		r.Header.Set("Authorization", "Bearer token123")

		// Test existing header
		value, exists := adapter.Header(r, "X-Custom-Header")
		if !exists {
			t.Error("Header 'X-Custom-Header' should exist")
		}
		if value != "custom-value" {
			t.Errorf("Header 'X-Custom-Header': got %q, want %q", value, "custom-value")
		}

		// Test non-existing header
		value, exists = adapter.Header(r, "Non-Existent")
		if exists {
			t.Error("Header 'Non-Existent' should not exist")
		}
		if value != "" {
			t.Errorf("Non-existing header: got %q, want empty string", value)
		}

		// Test case-insensitive header access
		value, exists = adapter.Header(r, "authorization")
		if !exists {
			t.Error("Header 'authorization' should exist (case-insensitive)")
		}
		if value != "Bearer token123" {
			t.Errorf("Header 'authorization': got %q, want %q", value, "Bearer token123")
		}
	})

	t.Run("Cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test", nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
		r.AddCookie(&http.Cookie{Name: "preferences", Value: "dark-mode"})

		// Test existing cookie
		value, exists := adapter.Cookie(r, "session")
		if !exists {
			t.Error("Cookie 'session' should exist")
		}
		if value != "abc123" {
			t.Errorf("Cookie 'session': got %q, want %q", value, "abc123")
		}

		// Test non-existing cookie
		value, exists = adapter.Cookie(r, "nonexistent")
		if exists {
			t.Error("Cookie 'nonexistent' should not exist")
		}
		if value != "" {
			t.Errorf("Non-existing cookie: got %q, want empty string", value)
		}

		// Test another existing cookie
		value, exists = adapter.Cookie(r, "preferences")
		if !exists {
			t.Error("Cookie 'preferences' should exist")
		}
		if value != "dark-mode" {
			t.Errorf("Cookie 'preferences': got %q, want %q", value, "dark-mode")
		}
	})

	t.Run("Path", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test", nil)

		// Test that Path panics as expected
		defer func() {
			if r := recover(); r == nil {
				t.Error("HTTPParameterAdapter.Path should panic")
			}
		}()
		adapter.Path(r, "id")
	})
}

// Benchmark tests for critical functions
func BenchmarkParseQueryParams(b *testing.B) {
	req := &queryTestReq{}
	r := httptest.NewRequest("GET", "/test?name=Alice&age=30&tags=foo&tags=bar", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseQueryParams(r, req)
	}
}

func BenchmarkSetFieldValue(b *testing.B) {
	type TestStruct struct {
		Value string
	}
	s := &TestStruct{}
	v := reflect.ValueOf(s).Elem()
	field := reflect.TypeOf(s).Elem().Field(0)
	fieldValue := v.Field(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setFieldValue(fieldValue, field, "test-value", []string{"test-value"})
	}
}

func BenchmarkGetFunctionName(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getFunctionName(DummyHandler)
	}
}

func BenchmarkParseOpenAPITag(b *testing.B) {
	tag := "api_key,in=query"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseOpenAPITag(tag)
	}
}
