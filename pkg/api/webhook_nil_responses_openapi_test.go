package api

import (
	"testing"
)

// TestAddStandardErrorResponsesForWebhookNilResponses tests the specific case where
// operation.Responses is nil to ensure 100% coverage of the addStandardErrorResponsesForWebhook function
func TestAddStandardErrorResponsesForWebhookNilResponses(t *testing.T) {
	t.Run("operation with nil responses", func(t *testing.T) {
		generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())
		components := &Components{
			Responses: map[string]*Response{},
			Schemas:   map[string]*Schema{},
		}

		// Create operation with explicitly nil Responses to hit the uncovered lines 1160-1162
		operation := &Operation{
			Responses: nil, // This is the key - nil responses
		}

		// This should trigger the nil check and initialization on lines 1160-1162
		generator.addStandardErrorResponsesForWebhook(operation, components)

		// Verify that Responses was initialized
		if operation.Responses == nil {
			t.Error("expected Responses to be initialized")
		}

		// Verify standard error responses were added
		if operation.Responses["422"] == nil {
			t.Error("expected 422 response to be added")
		}

		if operation.Responses["500"] == nil {
			t.Error("expected 500 response to be added")
		}

		// Should not have 400 response (skipped for webhooks)
		if operation.Responses["400"] != nil {
			t.Error("webhook should not have standard 400 response")
		}
	})

	t.Run("operation with existing responses map", func(t *testing.T) {
		generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())
		components := &Components{
			Responses: map[string]*Response{},
			Schemas:   map[string]*Schema{},
		}

		// Create operation with existing (non-nil) responses map
		operation := &Operation{
			Responses: map[string]*Response{
				"200": {Description: "Success"},
			},
		}

		// This should NOT trigger the nil initialization
		generator.addStandardErrorResponsesForWebhook(operation, components)

		// Verify existing response is preserved
		if operation.Responses["200"] == nil || operation.Responses["200"].Description != "Success" {
			t.Error("expected existing 200 response to be preserved")
		}

		// Verify standard error responses were added
		if operation.Responses["422"] == nil {
			t.Error("expected 422 response to be added")
		}

		if operation.Responses["500"] == nil {
			t.Error("expected 500 response to be added")
		}
	})
}
