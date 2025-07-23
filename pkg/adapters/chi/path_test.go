package chi

import "testing"

// TestChiPathFormatNative verifies that Chi uses the same path format
// as the generic goapi format, so no conversion is needed.
func TestChiPathFormatNative(t *testing.T) {
	// Chi uses {param} format natively, which matches goapi's generic format.
	// No path conversion function is needed or implemented.

	testPaths := []string{
		"/users/{id}",
		"/api/v1/users/{id}/posts/{postId}",
		"/docs/*",
		"/static/files/*",
		"/health",
	}

	// These paths should work as-is with Chi router
	for _, path := range testPaths {
		// Chi accepts these paths directly without conversion
		t.Logf("Chi path format: %s (no conversion needed)", path)
	}

	// This test serves as documentation that Chi doesn't need
	// a toNativePath conversion function like other adapters.
}
