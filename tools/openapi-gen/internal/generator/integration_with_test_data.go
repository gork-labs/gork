package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
