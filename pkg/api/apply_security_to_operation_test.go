package api

import (
	"testing"
)

// TestApplySecurityToOperationMissingCoverage tests specific edge cases to improve coverage
func TestApplySecurityToOperationMissingCoverage(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		route := &RouteInfo{
			Options: nil, // No options
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Should return early and not modify anything
		if len(op.Security) != 0 {
			t.Error("Expected no security when no options provided")
		}
	})

	t.Run("empty security list", func(t *testing.T) {
		route := &RouteInfo{
			Options: &HandlerOption{
				Security: []SecurityRequirement{}, // Empty security list
			},
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Should return early and not modify anything
		if len(op.Security) != 0 {
			t.Error("Expected no security when empty security list provided")
		}
	})

	t.Run("nil security schemes map", func(t *testing.T) {
		route := &RouteInfo{
			Options: &HandlerOption{
				Security: []SecurityRequirement{
					{Type: "basic"},
				},
			},
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: nil, // Nil map - should be initialized
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Should initialize the security schemes map
		if spec.Components.SecuritySchemes == nil {
			t.Error("Expected security schemes map to be initialized")
		}

		// Should have added BasicAuth scheme
		if _, exists := spec.Components.SecuritySchemes["BasicAuth"]; !exists {
			t.Error("Expected BasicAuth scheme to be added")
		}

		// Should have added security to operation
		if len(op.Security) != 1 {
			t.Errorf("Expected 1 security item, got %d", len(op.Security))
		}
	})

	t.Run("basic auth type", func(t *testing.T) {
		route := &RouteInfo{
			Options: &HandlerOption{
				Security: []SecurityRequirement{
					{Type: "basic"},
				},
			},
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Check BasicAuth scheme was added
		scheme, exists := spec.Components.SecuritySchemes["BasicAuth"]
		if !exists {
			t.Fatal("Expected BasicAuth scheme to be added")
		}

		if scheme.Type != "http" || scheme.Scheme != "basic" {
			t.Errorf("Expected http/basic scheme, got %s/%s", scheme.Type, scheme.Scheme)
		}

		// Check security was added to operation
		if len(op.Security) != 1 {
			t.Fatalf("Expected 1 security item, got %d", len(op.Security))
		}

		if _, exists := op.Security[0]["BasicAuth"]; !exists {
			t.Error("Expected BasicAuth in operation security")
		}
	})

	t.Run("bearer auth type", func(t *testing.T) {
		route := &RouteInfo{
			Options: &HandlerOption{
				Security: []SecurityRequirement{
					{Type: "bearer"},
				},
			},
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Check BearerAuth scheme was added
		scheme, exists := spec.Components.SecuritySchemes["BearerAuth"]
		if !exists {
			t.Fatal("Expected BearerAuth scheme to be added")
		}

		if scheme.Type != "http" || scheme.Scheme != "bearer" {
			t.Errorf("Expected http/bearer scheme, got %s/%s", scheme.Type, scheme.Scheme)
		}

		// Check security was added to operation
		if len(op.Security) != 1 {
			t.Fatalf("Expected 1 security item, got %d", len(op.Security))
		}

		if _, exists := op.Security[0]["BearerAuth"]; !exists {
			t.Error("Expected BearerAuth in operation security")
		}
	})

	t.Run("apiKey auth type", func(t *testing.T) {
		route := &RouteInfo{
			Options: &HandlerOption{
				Security: []SecurityRequirement{
					{Type: "apiKey"},
				},
			},
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Check ApiKeyAuth scheme was added
		scheme, exists := spec.Components.SecuritySchemes["ApiKeyAuth"]
		if !exists {
			t.Fatal("Expected ApiKeyAuth scheme to be added")
		}

		if scheme.Type != "apiKey" || scheme.In != "header" || scheme.Name != "X-API-Key" {
			t.Errorf("Expected apiKey/header/X-API-Key scheme, got %s/%s/%s", scheme.Type, scheme.In, scheme.Name)
		}

		// Check security was added to operation
		if len(op.Security) != 1 {
			t.Fatalf("Expected 1 security item, got %d", len(op.Security))
		}

		if _, exists := op.Security[0]["ApiKeyAuth"]; !exists {
			t.Error("Expected ApiKeyAuth in operation security")
		}
	})

	t.Run("unknown auth type", func(t *testing.T) {
		route := &RouteInfo{
			Options: &HandlerOption{
				Security: []SecurityRequirement{
					{Type: "unknown_type"}, // Unknown type - should be skipped
				},
			},
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Should not add any security schemes for unknown type
		if len(spec.Components.SecuritySchemes) != 0 {
			t.Error("Expected no security schemes for unknown type")
		}

		// Should not add security to operation
		if len(op.Security) != 0 {
			t.Error("Expected no security for unknown type")
		}
	})

	t.Run("multiple security types", func(t *testing.T) {
		route := &RouteInfo{
			Options: &HandlerOption{
				Security: []SecurityRequirement{
					{Type: "basic"},
					{Type: "bearer"},
					{Type: "apiKey"},
					{Type: "unknown"}, // Should be skipped
				},
			},
		}

		spec := &OpenAPISpec{
			Components: &Components{
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		}

		op := &Operation{}

		applySecurityToOperation(route, spec, op)

		// Should have 3 security schemes (unknown type skipped)
		if len(spec.Components.SecuritySchemes) != 3 {
			t.Errorf("Expected 3 security schemes, got %d", len(spec.Components.SecuritySchemes))
		}

		// Check all expected schemes exist
		expectedSchemes := []string{"BasicAuth", "BearerAuth", "ApiKeyAuth"}
		for _, schemeName := range expectedSchemes {
			if _, exists := spec.Components.SecuritySchemes[schemeName]; !exists {
				t.Errorf("Expected %s scheme to be added", schemeName)
			}
		}

		// Should have 3 security items in operation (unknown type skipped)
		if len(op.Security) != 3 {
			t.Errorf("Expected 3 security items in operation, got %d", len(op.Security))
		}
	})
}
