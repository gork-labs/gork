package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
