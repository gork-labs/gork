package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Mock adapter for testing convention parser
type mockConventionParameterAdapter struct {
	pathParams  map[string]string
	queryParams map[string]string
	headers     map[string]string
	cookies     map[string]string
}

func (m *mockConventionParameterAdapter) Path(r *http.Request, key string) (string, bool) {
	val, ok := m.pathParams[key]
	return val, ok
}

func (m *mockConventionParameterAdapter) Query(r *http.Request, key string) (string, bool) {
	val, ok := m.queryParams[key]
	return val, ok
}

func (m *mockConventionParameterAdapter) Header(r *http.Request, key string) (string, bool) {
	val, ok := m.headers[key]
	return val, ok
}

func (m *mockConventionParameterAdapter) Cookie(r *http.Request, key string) (string, bool) {
	val, ok := m.cookies[key]
	return val, ok
}

// Test request structures
type TestConventionRequest struct {
	Path struct {
		UserID string `gork:"user_id" validate:"required"`
		OrgID  int    `gork:"org_id" validate:"required"`
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
}

func TestConventionParser_ParseRequest(t *testing.T) {
	parser := NewConventionParser()

	tests := []struct {
		name     string
		adapter  *mockConventionParameterAdapter
		expected TestConventionRequest
		wantErr  bool
	}{
		{
			name: "successful parsing all sections",
			adapter: &mockConventionParameterAdapter{
				pathParams: map[string]string{
					"user_id": "user-123",
					"org_id":  "42",
				},
				queryParams: map[string]string{
					"limit":  "10",
					"fields": "name,email",
					"active": "true",
				},
				headers: map[string]string{
					"Authorization": "Bearer token123",
					"Content-Type":  "application/json",
				},
				cookies: map[string]string{
					"session_id": "sess-456",
				},
			},
			expected: TestConventionRequest{
				Path: struct {
					UserID string `gork:"user_id" validate:"required"`
					OrgID  int    `gork:"org_id" validate:"required"`
				}{
					UserID: "user-123",
					OrgID:  42,
				},
				Query: struct {
					Limit  int      `gork:"limit" validate:"min=1,max=100"`
					Fields []string `gork:"fields"`
					Active bool     `gork:"active"`
				}{
					Limit:  10,
					Fields: []string{"name", "email"},
					Active: true,
				},
				Headers: struct {
					Authorization string `gork:"Authorization" validate:"required"`
					ContentType   string `gork:"Content-Type"`
				}{
					Authorization: "Bearer token123",
					ContentType:   "application/json",
				},
				Cookies: struct {
					SessionID string `gork:"session_id"`
				}{
					SessionID: "sess-456",
				},
			},
			wantErr: false,
		},
		{
			name: "partial data - only path and query",
			adapter: &mockConventionParameterAdapter{
				pathParams: map[string]string{
					"user_id": "user-456",
				},
				queryParams: map[string]string{
					"limit": "5",
				},
			},
			expected: TestConventionRequest{
				Path: struct {
					UserID string `gork:"user_id" validate:"required"`
					OrgID  int    `gork:"org_id" validate:"required"`
				}{
					UserID: "user-456",
					OrgID:  0, // zero value
				},
				Query: struct {
					Limit  int      `gork:"limit" validate:"min=1,max=100"`
					Fields []string `gork:"fields"`
					Active bool     `gork:"active"`
				}{
					Limit:  5,
					Fields: nil,
					Active: false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{},
			}

			var result TestConventionRequest
			reqPtr := reflect.ValueOf(&result)

			err := parser.ParseRequest(context.Background(), req, reqPtr, tt.adapter)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check path section
				if result.Path.UserID != tt.expected.Path.UserID {
					t.Errorf("Path.UserID = %v, want %v", result.Path.UserID, tt.expected.Path.UserID)
				}
				if result.Path.OrgID != tt.expected.Path.OrgID {
					t.Errorf("Path.OrgID = %v, want %v", result.Path.OrgID, tt.expected.Path.OrgID)
				}

				// Check query section
				if result.Query.Limit != tt.expected.Query.Limit {
					t.Errorf("Query.Limit = %v, want %v", result.Query.Limit, tt.expected.Query.Limit)
				}
				if result.Query.Active != tt.expected.Query.Active {
					t.Errorf("Query.Active = %v, want %v", result.Query.Active, tt.expected.Query.Active)
				}

				// Check headers section
				if result.Headers.Authorization != tt.expected.Headers.Authorization {
					t.Errorf("Headers.Authorization = %v, want %v", result.Headers.Authorization, tt.expected.Headers.Authorization)
				}

				// Check cookies section
				if result.Cookies.SessionID != tt.expected.Cookies.SessionID {
					t.Errorf("Cookies.SessionID = %v, want %v", result.Cookies.SessionID, tt.expected.Cookies.SessionID)
				}
			}
		})
	}
}

func TestConventionParser_TypeParsing(t *testing.T) {
	parser := NewConventionParser()

	// Register a custom type parser for time.Time
	err := parser.RegisterTypeParser(func(ctx context.Context, value string) (*time.Time, error) {
		t, err := time.Parse(time.RFC3339, value)
		return &t, err
	})
	if err != nil {
		t.Fatalf("Failed to register type parser: %v", err)
	}

	type RequestWithCustomType struct {
		Query struct {
			Since time.Time `gork:"since"`
		}
	}

	adapter := &mockConventionParameterAdapter{
		queryParams: map[string]string{
			"since": "2023-01-01T00:00:00Z",
		},
	}

	req := &http.Request{URL: &url.URL{}}
	var result RequestWithCustomType
	reqPtr := reflect.ValueOf(&result)

	err = parser.ParseRequest(context.Background(), req, reqPtr, adapter)
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if !result.Query.Since.Equal(expected) {
		t.Errorf("Query.Since = %v, want %v", result.Query.Since, expected)
	}
}

func TestConventionParser_BodyParsing(t *testing.T) {
	parser := NewConventionParser()

	type RequestWithBody struct {
		Body struct {
			Name  string `gork:"name"`
			Email string `gork:"email"`
		}
	}

	jsonBody := `{"name":"John Doe","email":"john@example.com"}`
	req := &http.Request{
		Method: http.MethodPost,
		Body:   &nopCloser{strings.NewReader(jsonBody)},
	}

	var result RequestWithBody
	reqPtr := reflect.ValueOf(&result)

	err := parser.ParseRequest(context.Background(), req, reqPtr, nil)
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if result.Body.Name != "John Doe" {
		t.Errorf("Body.Name = %v, want %v", result.Body.Name, "John Doe")
	}
	if result.Body.Email != "john@example.com" {
		t.Errorf("Body.Email = %v, want %v", result.Body.Email, "john@example.com")
	}
}

func TestParseGorkTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected GorkTagInfo
	}{
		{
			tag: "field_name",
			expected: GorkTagInfo{
				Name: "field_name",
			},
		},
		{
			tag: "type,discriminator=email",
			expected: GorkTagInfo{
				Name:          "type",
				Discriminator: "email",
			},
		},
		{
			tag: "login_method,discriminator=oauth",
			expected: GorkTagInfo{
				Name:          "login_method",
				Discriminator: "oauth",
			},
		},
		{
			tag:      "",
			expected: GorkTagInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := parseGorkTag(tt.tag)
			if result.Name != tt.expected.Name {
				t.Errorf("Name = %v, want %v", result.Name, tt.expected.Name)
			}
			if result.Discriminator != tt.expected.Discriminator {
				t.Errorf("Discriminator = %v, want %v", result.Discriminator, tt.expected.Discriminator)
			}
		})
	}
}

// Helper for testing
type nopCloser struct {
	*strings.Reader
}

func (nopCloser) Close() error { return nil }

// Test setBasicFieldValue error paths
func TestConventionParser_SetBasicFieldValue_ErrorPaths(t *testing.T) {
	parser := NewConventionParser()

	tests := []struct {
		name    string
		field   reflect.StructField
		value   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid integer",
			field: reflect.StructField{
				Type: reflect.TypeOf(int(0)),
			},
			value:   "not-a-number",
			wantErr: true,
			errMsg:  "invalid integer value",
		},
		{
			name: "invalid unsigned integer",
			field: reflect.StructField{
				Type: reflect.TypeOf(uint(0)),
			},
			value:   "not-a-number",
			wantErr: true,
			errMsg:  "invalid unsigned integer value",
		},
		{
			name: "invalid boolean",
			field: reflect.StructField{
				Type: reflect.TypeOf(bool(false)),
			},
			value:   "not-a-bool",
			wantErr: true,
			errMsg:  "invalid boolean value",
		},
		{
			name: "invalid float",
			field: reflect.StructField{
				Type: reflect.TypeOf(float64(0)),
			},
			value:   "not-a-float",
			wantErr: true,
			errMsg:  "invalid float value",
		},
		{
			name: "invalid time format",
			field: reflect.StructField{
				Type: reflect.TypeOf(time.Time{}),
			},
			value:   "not-a-time",
			wantErr: true,
			errMsg:  "invalid time format",
		},
		{
			name: "unsupported type",
			field: reflect.StructField{
				Type: reflect.TypeOf(map[string]interface{}{}),
			},
			value:   "value",
			wantErr: true,
			errMsg:  "unsupported field type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a settable value of the field's type
			fieldValue := reflect.New(tt.field.Type).Elem()

			err := parser.setBasicFieldValue(fieldValue, tt.field, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setBasicFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("setBasicFieldValue() error = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// Test setSliceFieldValue error paths
func TestConventionParser_SetSliceFieldValue_ErrorPaths(t *testing.T) {
	parser := NewConventionParser()

	// Test with unsupported slice element type
	sliceType := reflect.TypeOf([]map[string]interface{}{})
	field := reflect.StructField{Type: sliceType}
	fieldValue := reflect.New(sliceType).Elem()

	err := parser.setSliceFieldValue(fieldValue, field, "value1,value2")
	if err == nil {
		t.Error("setSliceFieldValue() expected error for unsupported slice element type")
	}
	if !strings.Contains(err.Error(), "only string slices are supported") {
		t.Errorf("setSliceFieldValue() error = %v, want to contain 'only string slices are supported'", err.Error())
	}
}

// Test setSliceFieldValue with empty string
func TestConventionParser_SetSliceFieldValue_EmptyString(t *testing.T) {
	parser := NewConventionParser()

	// Test with empty string - should return nil and leave slice as zero value
	sliceType := reflect.TypeOf([]string{})
	field := reflect.StructField{Type: sliceType}
	fieldValue := reflect.New(sliceType).Elem()

	err := parser.setSliceFieldValue(fieldValue, field, "")
	if err != nil {
		t.Errorf("setSliceFieldValue() with empty string returned error: %v", err)
	}

	// Slice should remain as zero value (nil)
	if !fieldValue.IsNil() {
		t.Error("setSliceFieldValue() with empty string should leave slice as zero value")
	}
}

// Test parser error paths for coverage
func TestConventionParser_ErrorPaths(t *testing.T) {
	parser := NewConventionParser()

	// Test parseSection with invalid section value
	req := &http.Request{}
	adapter := &mockConventionParameterAdapter{}

	err := parser.parseSection(context.Background(), "query", reflect.Value{}, req, adapter)
	if err == nil {
		t.Error("parseSection() expected error for invalid value")
	}
}

// Test more complex error scenarios
func TestConventionParser_ComplexErrorScenarios(t *testing.T) {
	parser := NewConventionParser()
	req := &http.Request{}
	adapter := &mockConventionParameterAdapter{
		queryParams: map[string]string{
			"invalid_int":  "not-a-number",
			"invalid_bool": "not-a-bool",
		},
	}

	type TestErrorRequest struct {
		Query struct {
			InvalidInt  int  `gork:"invalid_int"`
			InvalidBool bool `gork:"invalid_bool"`
		}
	}

	request := &TestErrorRequest{}
	reqPtr := reflect.New(reflect.TypeOf(*request))

	// This should trigger error paths in setBasicFieldValue
	err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
	if err == nil {
		t.Error("ParseRequest() expected error for invalid type conversions")
	}
}

// Test parseBodySection edge cases for better coverage
func TestConventionParser_ParseBodySection_EdgeCases(t *testing.T) {
	parser := NewConventionParser()

	t.Run("GET request skips body parsing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		type TestBody struct {
			Name string `gork:"name"`
		}

		bodyValue := reflect.ValueOf(TestBody{})
		err := parser.parseBodySection(bodyValue, req)

		if err != nil {
			t.Errorf("parseBodySection() for GET should not error, got %v", err)
		}
	})

	t.Run("nil body is handled gracefully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Body = nil

		type TestBody struct {
			Name string `gork:"name"`
		}

		bodyValue := reflect.ValueOf(TestBody{})
		err := parser.parseBodySection(bodyValue, req)

		if err != nil {
			t.Errorf("parseBodySection() with nil body should not error, got %v", err)
		}
	})

	t.Run("empty body is handled gracefully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))

		type TestBody struct {
			Name string `gork:"name"`
		}

		bodyValue := reflect.ValueOf(TestBody{})
		err := parser.parseBodySection(bodyValue, req)

		if err != nil {
			t.Errorf("parseBodySection() with empty body should not error, got %v", err)
		}
	})

	t.Run("invalid JSON in body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"invalid": json}`))

		type TestBody struct {
			Name string `gork:"name"`
		}

		bodyValue := reflect.ValueOf(TestBody{})
		err := parser.parseBodySection(bodyValue, req)

		if err == nil {
			t.Error("parseBodySection() expected error for invalid JSON")
		}

		if !strings.Contains(err.Error(), "failed to decode JSON body") {
			t.Errorf("parseBodySection() error = %v, want to contain 'failed to decode JSON body'", err.Error())
		}
	})

	t.Run("error reading body", func(t *testing.T) {
		// Create a request with a body that will error when read
		req := httptest.NewRequest(http.MethodPost, "/test", &errorReader{})

		type TestBody struct {
			Name string `gork:"name"`
		}

		bodyValue := reflect.ValueOf(TestBody{})
		err := parser.parseBodySection(bodyValue, req)

		if err == nil {
			t.Error("parseBodySection() expected error from reading body")
		}

		if !strings.Contains(err.Error(), "failed to read request body") {
			t.Errorf("parseBodySection() error = %v, want to contain 'failed to read request body'", err.Error())
		}
	})
}

// errorReader implements io.Reader but always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

func (e *errorReader) Close() error {
	return nil
}

// Test all parser section methods for complete coverage
func TestConventionParser_AllSectionMethods(t *testing.T) {
	parser := NewConventionParser()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name": "test"}`))
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			"id": "123",
		},
		queryParams: map[string]string{
			"filter": "active",
		},
		headers: map[string]string{
			"Authorization": "Bearer token",
		},
		cookies: map[string]string{
			"session": "sess123",
		},
	}

	type CompleteTestRequest struct {
		Path struct {
			ID int `gork:"id"`
		}
		Query struct {
			Filter string `gork:"filter"`
		}
		Headers struct {
			Auth string `gork:"Authorization"`
		}
		Cookies struct {
			Session string `gork:"session"`
		}
		Body struct {
			Name string `gork:"name"`
		}
	}

	request := &CompleteTestRequest{}
	reqPtr := reflect.New(reflect.TypeOf(*request))

	err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
	if err != nil {
		t.Errorf("ParseRequest() error = %v, want nil", err)
	}

	result := reqPtr.Elem().Interface().(CompleteTestRequest)

	// Verify all sections were parsed correctly
	if result.Path.ID != 123 {
		t.Errorf("Path.ID = %d, want 123", result.Path.ID)
	}
	if result.Query.Filter != "active" {
		t.Errorf("Query.Filter = %s, want 'active'", result.Query.Filter)
	}
	if result.Headers.Auth != "Bearer token" {
		t.Errorf("Headers.Auth = %s, want 'Bearer token'", result.Headers.Auth)
	}
	if result.Cookies.Session != "sess123" {
		t.Errorf("Cookies.Session = %s, want 'sess123'", result.Cookies.Session)
	}
	if result.Body.Name != "test" {
		t.Errorf("Body.Name = %s, want 'test'", result.Body.Name)
	}
}

// Test parseSection error cases for complete coverage
func TestConventionParser_ParseSection_ErrorCases(t *testing.T) {
	parser := NewConventionParser()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	adapter := &mockConventionParameterAdapter{}

	t.Run("non-struct section value", func(t *testing.T) {
		// Pass an invalid reflect.Value (not a struct)
		invalidValue := reflect.ValueOf("not a struct")
		err := parser.parseSection(context.Background(), "Query", invalidValue, req, adapter)

		if err == nil {
			t.Error("parseSection() expected error for non-struct value")
		}
		if !strings.Contains(err.Error(), "section Query must be a struct") {
			t.Errorf("parseSection() error = %v, want to contain 'section Query must be a struct'", err.Error())
		}
	})

	t.Run("unknown section name", func(t *testing.T) {
		// Test with an unknown section name (should return nil)
		structValue := reflect.ValueOf(struct{}{})
		err := parser.parseSection(context.Background(), "UnknownSection", structValue, req, adapter)

		if err != nil {
			t.Errorf("parseSection() for unknown section should return nil, got error: %v", err)
		}
	})
}

// Test ParseRequest error cases for complete coverage
func TestConventionParser_ParseRequest_ErrorCases(t *testing.T) {
	parser := NewConventionParser()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	adapter := &mockConventionParameterAdapter{}

	t.Run("non-pointer request", func(t *testing.T) {
		// Pass a non-pointer value
		structValue := reflect.ValueOf(struct{}{})
		err := parser.ParseRequest(context.Background(), req, structValue, adapter)

		if err == nil {
			t.Error("ParseRequest() expected error for non-pointer value")
		}
		if !strings.Contains(err.Error(), "request must be a pointer to struct") {
			t.Errorf("ParseRequest() error = %v, want to contain 'request must be a pointer to struct'", err.Error())
		}
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		// Pass a pointer to non-struct
		stringVal := "not a struct"
		ptrValue := reflect.ValueOf(&stringVal)
		err := parser.ParseRequest(context.Background(), req, ptrValue, adapter)

		if err == nil {
			t.Error("ParseRequest() expected error for pointer to non-struct")
		}
		if !strings.Contains(err.Error(), "request must be a pointer to struct") {
			t.Errorf("ParseRequest() error = %v, want to contain 'request must be a pointer to struct'", err.Error())
		}
	})

	t.Run("section parsing error propagation", func(t *testing.T) {
		// Create a request struct with non-struct section to trigger error
		type BadRequest struct {
			Query string // Not a struct - should cause error
		}

		request := &BadRequest{}
		reqPtr := reflect.ValueOf(request)

		err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
		if err == nil {
			t.Error("ParseRequest() expected error for bad section type")
		}
		if !strings.Contains(err.Error(), "failed to parse Query section") {
			t.Errorf("ParseRequest() error = %v, want to contain 'failed to parse Query section'", err.Error())
		}
	})
}

// Test setFieldValue edge cases for complete coverage
func TestConventionParser_SetFieldValue_EdgeCases(t *testing.T) {
	parser := NewConventionParser()

	t.Run("custom type parser success", func(t *testing.T) {
		// Register a custom parser that should be called
		err := parser.RegisterTypeParser(func(ctx context.Context, value string) (*time.Time, error) {
			t, err := time.Parse("2006-01-02", value)
			return &t, err
		})
		if err != nil {
			t.Fatalf("Failed to register type parser: %v", err)
		}

		field := reflect.StructField{
			Type: reflect.TypeOf(time.Time{}),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err = parser.setFieldValue(context.Background(), fieldValue, field, "2023-01-01")
		if err != nil {
			t.Errorf("setFieldValue() with custom parser failed: %v", err)
		}

		expectedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		if !fieldValue.Interface().(time.Time).Equal(expectedTime) {
			t.Errorf("setFieldValue() result = %v, want %v", fieldValue.Interface(), expectedTime)
		}
	})

	t.Run("custom type parser error", func(t *testing.T) {
		// Register a custom parser that returns an error
		parser2 := NewConventionParser()
		err := parser2.RegisterTypeParser(func(ctx context.Context, value string) (*time.Time, error) {
			return nil, fmt.Errorf("custom parser error")
		})
		if err != nil {
			t.Fatalf("Failed to register type parser: %v", err)
		}

		field := reflect.StructField{
			Type: reflect.TypeOf(time.Time{}),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err = parser2.setFieldValue(context.Background(), fieldValue, field, "invalid")
		if err == nil {
			t.Error("setFieldValue() expected error from custom parser")
		}
		if !strings.Contains(err.Error(), "custom parser error") {
			t.Errorf("setFieldValue() error = %v, want to contain 'custom parser error'", err.Error())
		}
	})
}

// Test section parsing with nil adapter for complete coverage
func TestConventionParser_SectionParsing_NilAdapter(t *testing.T) {
	parser := NewConventionParser()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	type TestRequest struct {
		Path struct {
			ID string `gork:"id"`
		}
		Query struct {
			Filter string `gork:"filter"`
		}
		Headers struct {
			Auth string `gork:"Authorization"`
		}
		Cookies struct {
			Session string `gork:"session"`
		}
	}

	request := &TestRequest{}
	reqPtr := reflect.ValueOf(request)

	// Should not error with nil adapter - just skip parameter parsing
	err := parser.ParseRequest(context.Background(), req, reqPtr, nil)
	if err != nil {
		t.Errorf("ParseRequest() with nil adapter should not error, got: %v", err)
	}
}

// Test fields without gork tags for complete coverage
func TestConventionParser_FieldsWithoutGorkTags(t *testing.T) {
	parser := NewConventionParser()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			"id": "123",
		},
		queryParams: map[string]string{
			"filter": "active",
		},
		headers: map[string]string{
			"Authorization": "Bearer token",
		},
		cookies: map[string]string{
			"session": "sess123",
		},
	}

	type TestRequestWithoutTags struct {
		Path struct {
			ID         string `gork:"id"`
			NoTagField string // Field without gork tag - should be skipped
		}
		Query struct {
			Filter       string `gork:"filter"`
			AnotherField int    // Field without gork tag - should be skipped
		}
		Headers struct {
			Auth        string `gork:"Authorization"`
			NoTagHeader string // Field without gork tag - should be skipped
		}
		Cookies struct {
			Session     string `gork:"session"`
			NoTagCookie string // Field without gork tag - should be skipped
		}
	}

	request := &TestRequestWithoutTags{}
	reqPtr := reflect.ValueOf(request)

	err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
	if err != nil {
		t.Errorf("ParseRequest() should not error with fields without gork tags, got: %v", err)
	}

	// Verify only gork-tagged fields were set
	result := reqPtr.Elem().Interface().(TestRequestWithoutTags)
	if result.Path.ID != "123" {
		t.Errorf("Path.ID = %s, want '123'", result.Path.ID)
	}
	if result.Path.NoTagField != "" {
		t.Errorf("Path.NoTagField should remain empty, got '%s'", result.Path.NoTagField)
	}
	if result.Query.Filter != "active" {
		t.Errorf("Query.Filter = %s, want 'active'", result.Query.Filter)
	}
	if result.Query.AnotherField != 0 {
		t.Errorf("Query.AnotherField should remain zero, got %d", result.Query.AnotherField)
	}
}

// Test request struct with non-standard sections for complete coverage
func TestConventionParser_NonStandardSections(t *testing.T) {
	parser := NewConventionParser()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	adapter := &mockConventionParameterAdapter{}

	type TestRequestWithExtraFields struct {
		Path struct {
			ID string `gork:"id"`
		}
		Query struct {
			Filter string `gork:"filter"`
		}
		CustomSection string                 // Non-standard section - should be skipped
		AnotherField  map[string]interface{} // Non-standard section - should be skipped
	}

	request := &TestRequestWithExtraFields{}
	reqPtr := reflect.ValueOf(request)

	// Should not error and should skip non-standard sections
	err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
	if err != nil {
		t.Errorf("ParseRequest() should not error with non-standard sections, got: %v", err)
	}
}

// Test more setBasicFieldValue error cases for complete coverage
func TestConventionParser_SetBasicFieldValue_MoreErrorCases(t *testing.T) {
	parser := NewConventionParser()

	t.Run("negative value for unsigned int", func(t *testing.T) {
		field := reflect.StructField{
			Type: reflect.TypeOf(uint(0)),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err := parser.setBasicFieldValue(fieldValue, field, "-123")
		if err == nil {
			t.Error("setBasicFieldValue() expected error for negative uint")
		}
		if !strings.Contains(err.Error(), "invalid unsigned integer value") {
			t.Errorf("setBasicFieldValue() error = %v, want to contain 'invalid unsigned integer value'", err.Error())
		}
	})

	t.Run("very large float value", func(t *testing.T) {
		field := reflect.StructField{
			Type: reflect.TypeOf(float32(0)),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err := parser.setBasicFieldValue(fieldValue, field, "1e999") // Extremely large number
		if err == nil {
			t.Error("setBasicFieldValue() expected error for float overflow")
		}
		if !strings.Contains(err.Error(), "invalid float value") {
			t.Errorf("setBasicFieldValue() error = %v, want to contain 'invalid float value'", err.Error())
		}
	})

	t.Run("successful time.Time parsing", func(t *testing.T) {
		field := reflect.StructField{
			Type: reflect.TypeOf(time.Time{}),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err := parser.setBasicFieldValue(fieldValue, field, "2023-01-01T12:00:00Z")
		if err != nil {
			t.Errorf("setBasicFieldValue() for time.Time should not error, got: %v", err)
		}

		expectedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		actualTime := fieldValue.Interface().(time.Time)
		if !actualTime.Equal(expectedTime) {
			t.Errorf("setBasicFieldValue() time = %v, want %v", actualTime, expectedTime)
		}
	})

	t.Run("successful uint parsing", func(t *testing.T) {
		field := reflect.StructField{
			Type: reflect.TypeOf(uint64(0)),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err := parser.setBasicFieldValue(fieldValue, field, "12345")
		if err != nil {
			t.Errorf("setBasicFieldValue() for uint should not error, got: %v", err)
		}

		if fieldValue.Uint() != 12345 {
			t.Errorf("setBasicFieldValue() uint = %d, want 12345", fieldValue.Uint())
		}
	})

	t.Run("successful int parsing", func(t *testing.T) {
		field := reflect.StructField{
			Type: reflect.TypeOf(int64(0)),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err := parser.setBasicFieldValue(fieldValue, field, "-12345")
		if err != nil {
			t.Errorf("setBasicFieldValue() for int should not error, got: %v", err)
		}

		if fieldValue.Int() != -12345 {
			t.Errorf("setBasicFieldValue() int = %d, want -12345", fieldValue.Int())
		}
	})

	t.Run("successful float parsing", func(t *testing.T) {
		field := reflect.StructField{
			Type: reflect.TypeOf(float64(0)),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err := parser.setBasicFieldValue(fieldValue, field, "123.45")
		if err != nil {
			t.Errorf("setBasicFieldValue() for float should not error, got: %v", err)
		}

		if fieldValue.Float() != 123.45 {
			t.Errorf("setBasicFieldValue() float = %f, want 123.45", fieldValue.Float())
		}
	})

	t.Run("successful bool parsing", func(t *testing.T) {
		field := reflect.StructField{
			Type: reflect.TypeOf(bool(false)),
		}
		fieldValue := reflect.New(field.Type).Elem()

		err := parser.setBasicFieldValue(fieldValue, field, "true")
		if err != nil {
			t.Errorf("setBasicFieldValue() for bool should not error, got: %v", err)
		}

		if !fieldValue.Bool() {
			t.Error("setBasicFieldValue() bool should be true")
		}
	})
}

// Test section parsing with field setting errors for complete coverage
func TestConventionParser_SectionParsing_FieldErrors(t *testing.T) {
	parser := NewConventionParser()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	adapter := &mockConventionParameterAdapter{
		pathParams: map[string]string{
			"invalid_id": "not-a-number", // This will cause setFieldValue error
		},
		queryParams: map[string]string{
			"invalid_limit": "not-a-number", // This will cause setFieldValue error
		},
		headers: map[string]string{
			"X-Invalid-Count": "not-a-number", // This will cause setFieldValue error
		},
		cookies: map[string]string{
			"invalid_number": "not-a-number", // This will cause setFieldValue error
		},
	}

	// Test path section with field error
	t.Run("path section field error", func(t *testing.T) {
		type TestRequest struct {
			Path struct {
				InvalidID int `gork:"invalid_id"`
			}
		}

		request := &TestRequest{}
		reqPtr := reflect.ValueOf(request)

		err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
		if err == nil {
			t.Error("ParseRequest() expected error for invalid path parameter")
		}
		if !strings.Contains(err.Error(), "failed to set path parameter") {
			t.Errorf("ParseRequest() error = %v, want to contain 'failed to set path parameter'", err.Error())
		}
	})

	// Test query section with field error
	t.Run("query section field error", func(t *testing.T) {
		type TestRequest struct {
			Query struct {
				InvalidLimit int `gork:"invalid_limit"`
			}
		}

		request := &TestRequest{}
		reqPtr := reflect.ValueOf(request)

		err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
		if err == nil {
			t.Error("ParseRequest() expected error for invalid query parameter")
		}
		if !strings.Contains(err.Error(), "failed to set query parameter") {
			t.Errorf("ParseRequest() error = %v, want to contain 'failed to set query parameter'", err.Error())
		}
	})

	// Test headers section with field error
	t.Run("headers section field error", func(t *testing.T) {
		type TestRequest struct {
			Headers struct {
				InvalidCount int `gork:"X-Invalid-Count"`
			}
		}

		request := &TestRequest{}
		reqPtr := reflect.ValueOf(request)

		err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
		if err == nil {
			t.Error("ParseRequest() expected error for invalid header")
		}
		if !strings.Contains(err.Error(), "failed to set header") {
			t.Errorf("ParseRequest() error = %v, want to contain 'failed to set header'", err.Error())
		}
	})

	// Test cookies section with field error
	t.Run("cookies section field error", func(t *testing.T) {
		type TestRequest struct {
			Cookies struct {
				InvalidNumber int `gork:"invalid_number"`
			}
		}

		request := &TestRequest{}
		reqPtr := reflect.ValueOf(request)

		err := parser.ParseRequest(context.Background(), req, reqPtr, adapter)
		if err == nil {
			t.Error("ParseRequest() expected error for invalid cookie")
		}
		if !strings.Contains(err.Error(), "failed to set cookie") {
			t.Errorf("ParseRequest() error = %v, want to contain 'failed to set cookie'", err.Error())
		}
	})
}

// Test isBasicKind helper function
func TestConventionParser_IsBasicKind(t *testing.T) {
	parser := NewConventionParser()

	basicKinds := []reflect.Kind{
		reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Bool, reflect.Float32, reflect.Float64,
	}

	complexKinds := []reflect.Kind{
		reflect.Slice, reflect.Map, reflect.Struct, reflect.Interface, reflect.Chan,
		reflect.Func, reflect.Ptr, reflect.Array, reflect.Uintptr, reflect.Complex64,
	}

	for _, kind := range basicKinds {
		if !parser.isBasicKind(kind) {
			t.Errorf("isBasicKind(%v) = false, want true", kind)
		}
	}

	for _, kind := range complexKinds {
		if parser.isBasicKind(kind) {
			t.Errorf("isBasicKind(%v) = true, want false", kind)
		}
	}
}

// Test setBasicFieldValueForKind helper function
func TestConventionParser_SetBasicFieldValueForKind(t *testing.T) {
	parser := NewConventionParser()

	tests := []struct {
		name     string
		kind     reflect.Kind
		value    string
		setValue interface{}
		expected interface{}
		wantErr  bool
		errMsg   string
	}{
		{"string", reflect.String, "hello", "", "hello", false, ""},
		{"string_empty", reflect.String, "", "", "", false, ""},
		{"int", reflect.Int, "123", int(0), int64(123), false, ""},
		{"int_invalid", reflect.Int, "invalid", int(0), int64(0), true, "invalid integer value"},
		{"uint", reflect.Uint, "456", uint(0), uint64(456), false, ""},
		{"uint_invalid", reflect.Uint, "invalid", uint(0), uint64(0), true, "invalid unsigned integer value"},
		{"bool_true", reflect.Bool, "true", false, true, false, ""},
		{"bool_false", reflect.Bool, "false", true, false, false, ""},
		{"bool_invalid", reflect.Bool, "invalid", false, false, true, "invalid boolean value"},
		{"float32", reflect.Float32, "3.14", float32(0), float64(3.140000104904175), false, ""},
		{"float_invalid", reflect.Float32, "invalid", float32(0), float64(0), true, "invalid float value"},
		{"unsupported_kind", reflect.Slice, "test", []string{}, nil, true, "unsupported field type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.New(reflect.TypeOf(tt.setValue)).Elem()
			err := parser.setBasicFieldValueForKind(fieldValue, tt.kind, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setBasicFieldValueForKind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("setBasicFieldValueForKind() error = %v, want to contain %v", err.Error(), tt.errMsg)
				}
				return
			}

			if !tt.wantErr {
				switch tt.kind {
				case reflect.String:
					if fieldValue.String() != tt.expected.(string) {
						t.Errorf("setBasicFieldValueForKind() = %v, want %v", fieldValue.String(), tt.expected)
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if fieldValue.Int() != tt.expected.(int64) {
						t.Errorf("setBasicFieldValueForKind() = %v, want %v", fieldValue.Int(), tt.expected)
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					if fieldValue.Uint() != tt.expected.(uint64) {
						t.Errorf("setBasicFieldValueForKind() = %v, want %v", fieldValue.Uint(), tt.expected)
					}
				case reflect.Bool:
					if fieldValue.Bool() != tt.expected.(bool) {
						t.Errorf("setBasicFieldValueForKind() = %v, want %v", fieldValue.Bool(), tt.expected)
					}
				case reflect.Float32, reflect.Float64:
					if fieldValue.Float() != tt.expected.(float64) {
						t.Errorf("setBasicFieldValueForKind() = %v, want %v", fieldValue.Float(), tt.expected)
					}
				}
			}
		})
	}
}
