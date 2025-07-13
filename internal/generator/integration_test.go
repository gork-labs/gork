package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullGenerationIntegration(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string
		expected    func(*testing.T, *OpenAPISpec)
		expectError bool
	}{
		{
			name: "simple API with basic types",
			setupFiles: map[string]string{
				"models/user.go": `
package models

import "time"

// User represents a system user
type User struct {
	// Unique identifier
	ID string ` + "`json:\"id\" validate:\"required,uuid\"`" + `
	
	// Email address
	Email string ` + "`json:\"email\" validate:\"required,email,max=255\"`" + `
	
	// Username
	Username string ` + "`json:\"username\" validate:\"required,alphanum,min=3,max=50\"`" + `
	
	// Age in years
	Age *int ` + "`json:\"age,omitempty\" validate:\"omitempty,gte=0,lte=150\"`" + `
	
	// Created timestamp
	CreatedAt time.Time ` + "`json:\"createdAt\"`" + `
}

// CreateUserRequest for creating a new user
type CreateUserRequest struct {
	Email    string ` + "`json:\"email\" validate:\"required,email\"`" + `
	Username string ` + "`json:\"username\" validate:\"required,alphanum,min=3,max=50\"`" + `
	Age      *int   ` + "`json:\"age,omitempty\" validate:\"omitempty,gte=0,lte=150\"`" + `
}

// UserResponse standard user response
type UserResponse struct {
	User *User ` + "`json:\"user\"`" + `
}

// UserListResponse for listing users
type UserListResponse struct {
	Users []User ` + "`json:\"users\"`" + `
	Total int    ` + "`json:\"total\"`" + `
}
`,
				"handlers/user_handlers.go": `
package handlers

import (
	"context"
	"../models"
)

// ListUsers returns all users
func ListUsers(ctx context.Context, req ListUsersRequest) (*models.UserListResponse, error) {
	return nil, nil
}

// GetUser returns a user by ID
func GetUser(ctx context.Context, req GetUserRequest) (*models.UserResponse, error) {
	return nil, nil
}

// CreateUser creates a new user
func CreateUser(ctx context.Context, req models.CreateUserRequest) (*models.UserResponse, error) {
	return nil, nil
}

// UpdateUser updates an existing user
func UpdateUser(ctx context.Context, req UpdateUserRequest) (*models.UserResponse, error) {
	return nil, nil
}

// DeleteUser deletes a user
func DeleteUser(ctx context.Context, req DeleteUserRequest) (*models.UserResponse, error) {
	return nil, nil
}

// Request types
type ListUsersRequest struct {
	Page     int    ` + "`json:\"page,omitempty\" validate:\"omitempty,min=1\"`" + `
	PageSize int    ` + "`json:\"pageSize,omitempty\" validate:\"omitempty,min=1,max=100\"`" + `
	Search   string ` + "`json:\"search,omitempty\" validate:\"omitempty,max=255\"`" + `
}

type GetUserRequest struct {
	ID string ` + "`json:\"id\" validate:\"required,uuid\"`" + `
}

type UpdateUserRequest struct {
	ID       string ` + "`json:\"id\" validate:\"required,uuid\"`" + `
	Email    string ` + "`json:\"email\" validate:\"required,email\"`" + `
	Username string ` + "`json:\"username\" validate:\"required,alphanum,min=3,max=50\"`" + `
	Age      *int   ` + "`json:\"age,omitempty\" validate:\"omitempty,gte=0,lte=150\"`" + `
}

type DeleteUserRequest struct {
	ID string ` + "`json:\"id\" validate:\"required,uuid\"`" + `
}
`,
				"routes/routes.go": `
package routes

import (
	"net/http"
	"../handlers"
	"github.com/example/openapi-gen/pkg/api"
)

func SetupRoutes() {
	http.HandleFunc("GET /users", api.HandlerFunc(handlers.ListUsers))
	http.HandleFunc("GET /users/{id}", api.HandlerFunc(handlers.GetUser))
	http.HandleFunc("POST /users", api.HandlerFunc(handlers.CreateUser))
	http.HandleFunc("PUT /users/{id}", api.HandlerFunc(handlers.UpdateUser))
	http.HandleFunc("DELETE /users/{id}", api.HandlerFunc(handlers.DeleteUser))
}
`,
			},
			expected: func(t *testing.T, spec *OpenAPISpec) {
				// Should have OpenAPI 3.1.0
				assert.Equal(t, "3.1.0", spec.OpenAPI)
				
				// Should have paths
				assert.Contains(t, spec.Paths, "/users")
				assert.Contains(t, spec.Paths, "/users/{id}")
				
				// Check GET /users
				usersPath := spec.Paths["/users"]
				assert.NotNil(t, usersPath.Get)
				assert.NotNil(t, usersPath.Post)
				
				// Check GET /users/{id}
				userByIdPath := spec.Paths["/users/{id}"]
				assert.NotNil(t, userByIdPath.Get)
				assert.NotNil(t, userByIdPath.Put)
				assert.NotNil(t, userByIdPath.Delete)
				
				// Should have schemas
				assert.Contains(t, spec.Components.Schemas, "User")
				assert.Contains(t, spec.Components.Schemas, "CreateUserRequest")
				assert.Contains(t, spec.Components.Schemas, "UserResponse")
				assert.Contains(t, spec.Components.Schemas, "UserListResponse")
				
				// Check User schema
				userSchema := spec.Components.Schemas["User"]
				assert.Equal(t, "object", userSchema.Type)
				assert.Contains(t, userSchema.Required, "id")
				assert.Contains(t, userSchema.Required, "email")
				assert.Contains(t, userSchema.Required, "username")
				assert.NotContains(t, userSchema.Required, "age") // Optional field
				
				// Check field validations
				emailProp := userSchema.Properties["email"]
				assert.Equal(t, "email", emailProp.Format)
				assert.Equal(t, 255, *emailProp.MaxLength)
				
				usernameProp := userSchema.Properties["username"]
				assert.Equal(t, `^[a-zA-Z0-9]+$`, usernameProp.Pattern)
				assert.Equal(t, 3, *usernameProp.MinLength)
				assert.Equal(t, 50, *usernameProp.MaxLength)
				
				ageProp := userSchema.Properties["age"]
				assert.True(t, ageProp.Nullable)
				assert.Equal(t, 0.0, *ageProp.Minimum)
				assert.Equal(t, 150.0, *ageProp.Maximum)
			},
		},
		{
			name: "API with union types",
			setupFiles: map[string]string{
				"models/payment.go": `
package models

// Since we're in a test environment, we'll test with inline union patterns
// that the parser can detect without resolving imports

// PaymentOptions for OneOf flat JSON (all pointer fields, no JSON tags)
type PaymentOptions struct {
	CreditCard *CreditCardPayment
	BankACH    *BankACHPayment
	PayPal     *PayPalPayment
}

// NestedPaymentOptions for OneOf nested JSON (with JSON tags)
type NestedPaymentOptions struct {
	CreditCard *CreditCardPayment ` + "`json:\"creditCard,omitempty\"`" + `
	BankACH    *BankACHPayment    ` + "`json:\"bankACH,omitempty\"`" + `
	PayPal     *PayPalPayment     ` + "`json:\"payPal,omitempty\"`" + `
}

// Payment method types
type CreditCardPayment struct {
	CardNumber string ` + "`json:\"cardNumber\" validate:\"required,len=16,numeric\"`" + `
	CVV        string ` + "`json:\"cvv\" validate:\"required,len=3,numeric\"`" + `
	ExpMonth   int    ` + "`json:\"expMonth\" validate:\"required,min=1,max=12\"`" + `
	ExpYear    int    ` + "`json:\"expYear\" validate:\"required,min=2024,max=2040\"`" + `
}

type BankACHPayment struct {
	AccountNumber string ` + "`json:\"accountNumber\" validate:\"required,min=8,max=17,numeric\"`" + `
	RoutingNumber string ` + "`json:\"routingNumber\" validate:\"required,len=9,numeric\"`" + `
	AccountType   string ` + "`json:\"accountType\" validate:\"required,oneof=checking savings\"`" + `
}

type PayPalPayment struct {
	Email         string ` + "`json:\"email\" validate:\"required,email\"`" + `
	PayPalAccount string ` + "`json:\"paypalAccount,omitempty\"`" + `
}

`,
				"handlers/payment_handlers.go": `
package handlers

import "context"

// ProcessPayment processes a payment with flat structure
func ProcessPayment(ctx context.Context, req PaymentOptions) (*PaymentResponse, error) {
	return nil, nil
}

// ProcessNestedPayment processes a payment with nested structure
func ProcessNestedPayment(ctx context.Context, req NestedPaymentOptions) (*PaymentResponse, error) {
	return nil, nil
}

type PaymentResponse struct {
	TransactionID string ` + "`json:\"transactionId\"`" + `
	Status        string ` + "`json:\"status\"`" + `
}
`,
			},
			expected: func(t *testing.T, spec *OpenAPISpec) {
				// Should have the component schemas for payment types
				assert.Contains(t, spec.Components.Schemas, "PaymentOptions")
				assert.Contains(t, spec.Components.Schemas, "NestedPaymentOptions")
				assert.Contains(t, spec.Components.Schemas, "CreditCardPayment")
				assert.Contains(t, spec.Components.Schemas, "BankACHPayment")
				assert.Contains(t, spec.Components.Schemas, "PayPalPayment")
				
				// Check PaymentOptions - should be detected as union options type (all pointer fields, no JSON tags)
				paymentOptionsSchema := spec.Components.Schemas["PaymentOptions"]
				assert.NotNil(t, paymentOptionsSchema)
				assert.Equal(t, "object", paymentOptionsSchema.Type)
				
				// Check NestedPaymentOptions - regular object with JSON tags
				nestedPaymentOptionsSchema := spec.Components.Schemas["NestedPaymentOptions"]
				assert.NotNil(t, nestedPaymentOptionsSchema)
				assert.Equal(t, "object", nestedPaymentOptionsSchema.Type)
				assert.Contains(t, nestedPaymentOptionsSchema.Properties, "creditCard")
				assert.Contains(t, nestedPaymentOptionsSchema.Properties, "bankACH")
				assert.Contains(t, nestedPaymentOptionsSchema.Properties, "payPal")
				
				// Should have all component schemas
				assert.Contains(t, spec.Components.Schemas, "CreditCardPayment")
				assert.Contains(t, spec.Components.Schemas, "BankACHPayment")
				assert.Contains(t, spec.Components.Schemas, "PayPalPayment")
			},
		},
		{
			name: "API with custom validators",
			setupFiles: map[string]string{
				"models/custom.go": `
package models

// CustomValidatedUser with custom validation tags
type CustomValidatedUser struct {
	ID       string ` + "`json:\"id\" validate:\"required,uuid\"`" + `
	Username string ` + "`json:\"username\" validate:\"required,username,min=3,max=50\"`" + `
	Password string ` + "`json:\"password\" validate:\"required,strongpassword\"`" + `
	Profile  string ` + "`json:\"profile\" validate:\"required,profilename\"`" + `
}
`,
				"validators/custom_validators.go": `
package validators

import "github.com/go-playground/validator/v10"

// Custom validator functions
func ValidateUsername(fl validator.FieldLevel) bool {
	return true // Mock implementation
}

func ValidateStrongPassword(fl validator.FieldLevel) bool {
	return true // Mock implementation  
}

func ValidateProfileName(fl validator.FieldLevel) bool {
	return true // Mock implementation
}
`,
			},
			expected: func(t *testing.T, spec *OpenAPISpec) {
				// Should have the custom validated schema
				assert.Contains(t, spec.Components.Schemas, "CustomValidatedUser")
				
				userSchema := spec.Components.Schemas["CustomValidatedUser"]
				
				// Custom validators should appear in descriptions
				usernameProp := userSchema.Properties["username"]
				assert.Contains(t, usernameProp.Description, "Custom username validation")
				
				passwordProp := userSchema.Properties["password"]
				assert.Contains(t, passwordProp.Description, "Strong password requirements")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir := t.TempDir()
			
			// Create files
			for filePath, content := range tt.setupFiles {
				fullPath := filepath.Join(tempDir, filePath)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}
			
			// Create generator and parse directory
			gen := New("Test API", "1.0.0")
			
			// Register custom validators if test includes them
			if _, hasCustom := tt.setupFiles["validators/custom_validators.go"]; hasCustom {
				gen.RegisterCustomValidator("username", "Custom username validation")
				gen.RegisterCustomValidator("strongpassword", "Strong password requirements")
				gen.RegisterCustomValidator("profilename", "Profile name validation")
			}
			
			// Parse all subdirectories
			dirsToScan := []string{tempDir}
			handlerDir := filepath.Join(tempDir, "handlers")
			if info, err := os.Stat(handlerDir); err == nil && info.IsDir() {
				dirsToScan = append(dirsToScan, handlerDir)
			}
			modelDir := filepath.Join(tempDir, "models")
			if info, err := os.Stat(modelDir); err == nil && info.IsDir() {
				dirsToScan = append(dirsToScan, modelDir)
			}
			
			err := gen.ParseDirectories(dirsToScan)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			
			// Parse routes if routes file exists
			routeFile := filepath.Join(tempDir, "routes", "routes.go")
			if _, err := os.Stat(routeFile); err == nil {
				err = gen.ParseRoutes([]string{routeFile})
				require.NoError(t, err)
			}
			
			// Generate OpenAPI spec
			spec := gen.Generate()
			require.NotNil(t, spec)
			
			// Run expectations
			tt.expected(t, spec)
		})
	}
}

func TestGeneratorWithRealExamples(t *testing.T) {
	// Test with the actual examples directory
	gen := New("Example API", "1.0.0")
	
	// Parse all subdirectories in examples
	err := gen.ParseDirectories([]string{
		"../../examples",
		"../../examples/handlers",
	})
	require.NoError(t, err)
	
	// Parse routes
	err = gen.ParseRoutes([]string{"../../examples/routes.go"})
	require.NoError(t, err)
	
	// Generate spec
	spec := gen.Generate()
	require.NotNil(t, spec)
	
	// Basic validations
	assert.Equal(t, "3.1.0", spec.OpenAPI)
	assert.Equal(t, "Example API", spec.Info.Title)
	assert.Equal(t, "1.0.0", spec.Info.Version)
	
	// Should have paths
	assert.NotEmpty(t, spec.Paths)
	
	// Should have schemas
	assert.NotEmpty(t, spec.Components.Schemas)
	
	// Check for expected schemas from examples
	expectedSchemas := []string{
		"LoginRequest",
		"LoginResponse",
		"UserResponse",
		"CreateUserRequest",
		"UpdateUserRequest",
		"GetUserRequest",
		"DeleteUserRequest",
		"ListUsersRequest",
		"ListUsersResponse",
		"AdminUserResponse",
		"UpdateUserPaymentMethodRequest",
		"UpdateUserPreferencesRequest",
		"PaymentMethodOptions",
		"CreditCardPaymentMethod",
	}
	
	// Check that we have at least some of the expected schemas
	foundCount := 0
	for _, schemaName := range expectedSchemas {
		if _, exists := spec.Components.Schemas[schemaName]; exists {
			t.Logf("Found expected schema: %s", schemaName)
			foundCount++
		}
	}
	
	// We should find at least half of the expected schemas
	assert.GreaterOrEqual(t, foundCount, len(expectedSchemas)/2, "Should find at least half of expected schemas")
}

func TestGeneratorWithTestData(t *testing.T) {
	// Test with the testdata directory
	gen := New("TestData API", "1.0.0")
	
	// Parse all subdirectories in testdata
	err := gen.ParseDirectories([]string{
		"../../testdata",
		"../../testdata/handlers",
	})
	require.NoError(t, err)
	
	// Parse routes
	err = gen.ParseRoutes([]string{"../../testdata/routes.go"})
	require.NoError(t, err)
	
	// Generate spec
	spec := gen.Generate()
	require.NotNil(t, spec)
	
	// Basic validations
	assert.Equal(t, "3.1.0", spec.OpenAPI)
	assert.Equal(t, "TestData API", spec.Info.Title)
	assert.Equal(t, "1.0.0", spec.Info.Version)
	
	// Should have paths
	assert.NotEmpty(t, spec.Paths)
	
	// Check for expected paths
	expectedPaths := []string{
		"/basic/hello",
		"/basic/hello-with-query",
		"/basic/hello-with-json",
		"/basic/hello-with-json-and-query",
		"/unions/any-of-without-wrapper",
		"/unions/any-of-union2",
	}
	
	for _, path := range expectedPaths {
		assert.Contains(t, spec.Paths, path, "Expected path %s not found", path)
	}
	
	// Should have schemas
	assert.NotEmpty(t, spec.Components.Schemas)
	
	// Check for expected schemas from testdata
	expectedSchemas := []string{
		"GetWithoutQueryParamsReq",
		"GetWithoutQueryParamsResp",
		"GetWithQueryParamsReq",
		"GetWithQueryParamsResp",
		"PostWithJsonBodyReq",
		"PostWithJsonBodyResp",
		"PostWithJsonBodyAndQueryParamsReq",
		"PostWithJsonBodyAndQueryParamsResp",
		"Option1",
		"Option2",
		"PaymentOptions",
		"AnyOfWithoutWrapperReq",
		"BodyWithoutWrapperResp",
		"AnyOfUnion2",
		"BodyWithWrapperResp",
	}
	
	// Check that we have at least most of the expected schemas
	foundCount := 0
	for _, schemaName := range expectedSchemas {
		if _, exists := spec.Components.Schemas[schemaName]; exists {
			t.Logf("Found expected schema: %s", schemaName)
			foundCount++
		} else {
			t.Logf("Missing expected schema: %s", schemaName)
		}
	}
	
	// We should find most of the expected schemas
	assert.GreaterOrEqual(t, foundCount, len(expectedSchemas)*3/4, "Should find at least 3/4 of expected schemas")
}

func TestParseDirectoryErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string
		expectError bool
		errorText   string
	}{
		{
			name: "invalid Go syntax",
			setupFiles: map[string]string{
				"invalid.go": `
package invalid

type User struct {
	ID string ` + "`json:\"id\"" + ` // Missing closing backtick
}
`,
			},
			expectError: true,
		},
		{
			name: "empty directory",
			setupFiles: map[string]string{
				"empty.txt": "not a go file",
			},
			expectError: false, // Should not error on non-Go files
		},
		{
			name: "non-existent directory",
			setupFiles: map[string]string{}, // Empty, will test non-existent dir
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "non-existent directory" {
				gen := New("Test API", "1.0.0")
				err := gen.ParseDirectories([]string{"/path/that/does/not/exist"})
				assert.Error(t, err)
				return
			}
			
			// Create temporary directory structure
			tempDir := t.TempDir()
			
			// Create files
			for filePath, content := range tt.setupFiles {
				fullPath := filepath.Join(tempDir, filePath)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}
			
			// Test parsing
			gen := New("Test API", "1.0.0")
			err := gen.ParseDirectories([]string{tempDir})
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSchemaGeneration(t *testing.T) {
	tests := []struct {
		name       string
		structCode string
		expected   func(*testing.T, map[string]*Schema)
	}{
		{
			name: "basic struct with all field types",
			structCode: `
package models

import "time"

type TestStruct struct {
	// String fields
	Name        string  ` + "`json:\"name\" validate:\"required,max=100\"`" + `
	Description *string ` + "`json:\"description,omitempty\" validate:\"omitempty,max=500\"`" + `
	
	// Numeric fields
	Age    int     ` + "`json:\"age\" validate:\"gte=0,lte=150\"`" + `
	Score  float64 ` + "`json:\"score\" validate:\"gte=0,lte=100\"`" + `
	Points *int    ` + "`json:\"points,omitempty\" validate:\"omitempty,gte=0\"`" + `
	
	// Boolean
	Active bool ` + "`json:\"active\"`" + `
	
	// Arrays and slices
	Tags    []string ` + "`json:\"tags,omitempty\" validate:\"omitempty,dive,alphanum\"`" + `
	Numbers []int    ` + "`json:\"numbers,omitempty\" validate:\"omitempty,min=1,max=10\"`" + `
	
	// Time
	CreatedAt time.Time  ` + "`json:\"createdAt\"`" + `
	UpdatedAt *time.Time ` + "`json:\"updatedAt,omitempty\"`" + `
	
	// Embedded/nested
	Profile UserProfile ` + "`json:\"profile\"`" + `
	
	// Map
	Metadata map[string]interface{} ` + "`json:\"metadata,omitempty\"`" + `
}

type UserProfile struct {
	Bio      string ` + "`json:\"bio,omitempty\" validate:\"omitempty,max=1000\"`" + `
	Location string ` + "`json:\"location,omitempty\" validate:\"omitempty,max=100\"`" + `
}
`,
			expected: func(t *testing.T, schemas map[string]*Schema) {
				require.Contains(t, schemas, "TestStruct")
				require.Contains(t, schemas, "UserProfile")
				
				testSchema := schemas["TestStruct"]
				assert.Equal(t, "object", testSchema.Type)
				
				// Check required fields
				assert.Contains(t, testSchema.Required, "name")
				assert.NotContains(t, testSchema.Required, "description") // Optional pointer
				assert.NotContains(t, testSchema.Required, "points")      // Optional pointer
				
				// Check string properties
				nameProp := testSchema.Properties["name"]
				assert.Equal(t, "string", nameProp.Type)
				assert.Equal(t, 100, *nameProp.MaxLength)
				
				descProp := testSchema.Properties["description"]
				assert.Equal(t, "string", descProp.Type)
				assert.True(t, descProp.Nullable)
				assert.Equal(t, 500, *descProp.MaxLength)
				
				// Check numeric properties
				ageProp := testSchema.Properties["age"]
				assert.Equal(t, "integer", ageProp.Type)
				assert.Equal(t, 0.0, *ageProp.Minimum)
				assert.Equal(t, 150.0, *ageProp.Maximum)
				
				scoreProp := testSchema.Properties["score"]
				assert.Equal(t, "number", scoreProp.Type)
				assert.Equal(t, 0.0, *scoreProp.Minimum)
				assert.Equal(t, 100.0, *scoreProp.Maximum)
				
				// Check array properties
				tagsProp := testSchema.Properties["tags"]
				assert.Equal(t, "array", tagsProp.Type)
				assert.Equal(t, "string", tagsProp.Items.Type)
				assert.Equal(t, `^[a-zA-Z0-9]+$`, tagsProp.Items.Pattern)
				
				numbersProp := testSchema.Properties["numbers"]
				assert.Equal(t, "array", numbersProp.Type)
				assert.Equal(t, "integer", numbersProp.Items.Type)
				assert.Equal(t, 1, *numbersProp.MinItems)
				assert.Equal(t, 10, *numbersProp.MaxItems)
				
				// Check time properties
				createdProp := testSchema.Properties["createdAt"]
				assert.Equal(t, "string", createdProp.Type)
				assert.Equal(t, "date-time", createdProp.Format)
				
				updatedProp := testSchema.Properties["updatedAt"]
				assert.Equal(t, "string", updatedProp.Type)
				assert.Equal(t, "date-time", updatedProp.Format)
				assert.True(t, updatedProp.Nullable)
				
				// Check nested object
				profileProp := testSchema.Properties["profile"]
				assert.Equal(t, "#/components/schemas/UserProfile", profileProp.Ref)
				
				// Check map property
				metadataProp := testSchema.Properties["metadata"]
				assert.Equal(t, "object", metadataProp.Type)
				assert.NotNil(t, metadataProp.AdditionalProperties)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test.go")
			
			err := os.WriteFile(filePath, []byte(tt.structCode), 0644)
			require.NoError(t, err)
			
			// Parse with generator
			gen := New("Test API", "1.0.0")
			err = gen.ParseDirectories([]string{tempDir})
			require.NoError(t, err)
			
			// Generate the spec to trigger schema generation
			spec := gen.Generate()
			require.NotNil(t, spec)
			
			// Get generated schemas
			tt.expected(t, spec.Components.Schemas)
		})
	}
}