package api

import (
	"testing"
)

func TestAttachOperation(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		expectedField string
	}{
		{
			name:          "GET method",
			method:        "get",
			expectedField: "Get",
		},
		{
			name:          "POST method",
			method:        "post",
			expectedField: "Post",
		},
		{
			name:          "PUT method",
			method:        "put",
			expectedField: "Put",
		},
		{
			name:          "PATCH method",
			method:        "patch",
			expectedField: "Patch",
		},
		{
			name:          "DELETE method",
			method:        "delete",
			expectedField: "Delete",
		},
		{
			name:          "unsupported method",
			method:        "options",
			expectedField: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh PathItem for each test
			pathItem := &PathItem{}
			operation := &Operation{
				OperationID: "test-operation",
			}

			// Call attachOperation
			attachOperation(pathItem, tt.method, operation)

			// Check which field was set
			var setOperation *Operation
			var fieldName string

			if pathItem.Get != nil {
				setOperation = pathItem.Get
				fieldName = "Get"
			} else if pathItem.Post != nil {
				setOperation = pathItem.Post
				fieldName = "Post"
			} else if pathItem.Put != nil {
				setOperation = pathItem.Put
				fieldName = "Put"
			} else if pathItem.Patch != nil {
				setOperation = pathItem.Patch
				fieldName = "Patch"
			} else if pathItem.Delete != nil {
				setOperation = pathItem.Delete
				fieldName = "Delete"
			}

			if tt.expectedField == "" {
				// Unsupported method should not set any field
				if setOperation != nil {
					t.Errorf("Expected no operation to be set for unsupported method %s, but %s was set", tt.method, fieldName)
				}
			} else {
				// Supported method should set the correct field
				if setOperation == nil {
					t.Errorf("Expected %s field to be set for method %s, but no operation was set", tt.expectedField, tt.method)
				} else if fieldName != tt.expectedField {
					t.Errorf("Expected %s field to be set for method %s, but %s was set instead", tt.expectedField, tt.method, fieldName)
				} else if setOperation.OperationID != "test-operation" {
					t.Errorf("Expected operation to be attached correctly, but OperationID = %s", setOperation.OperationID)
				}
			}
		})
	}
}

func TestAttachOperationMultipleCalls(t *testing.T) {
	// Test that multiple operations can be attached to the same PathItem
	pathItem := &PathItem{}

	getOp := &Operation{OperationID: "get-operation"}
	postOp := &Operation{OperationID: "post-operation"}
	putOp := &Operation{OperationID: "put-operation"}

	attachOperation(pathItem, "get", getOp)
	attachOperation(pathItem, "post", postOp)
	attachOperation(pathItem, "put", putOp)

	if pathItem.Get == nil || pathItem.Get.OperationID != "get-operation" {
		t.Error("GET operation not attached correctly")
	}
	if pathItem.Post == nil || pathItem.Post.OperationID != "post-operation" {
		t.Error("POST operation not attached correctly")
	}
	if pathItem.Put == nil || pathItem.Put.OperationID != "put-operation" {
		t.Error("PUT operation not attached correctly")
	}

	// Ensure other methods are still nil
	if pathItem.Patch != nil {
		t.Error("PATCH should be nil when not set")
	}
	if pathItem.Delete != nil {
		t.Error("DELETE should be nil when not set")
	}
}

func TestEnsureStdResponses(t *testing.T) {
	t.Run("initialize responses on nil map", func(t *testing.T) {
		comps := &Components{
			Responses: nil,
		}

		ensureStdResponses(comps)

		if comps.Responses == nil {
			t.Fatal("Expected Responses map to be initialized")
		}

		// Check that all standard responses are added
		expectedResponses := []string{"BadRequest", "UnprocessableEntity", "InternalServerError"}
		for _, name := range expectedResponses {
			if _, exists := comps.Responses[name]; !exists {
				t.Errorf("Expected response '%s' to be added", name)
			}
		}
	})

	t.Run("add responses to existing map", func(t *testing.T) {
		comps := &Components{
			Responses: map[string]*Response{
				"CustomResponse": {Description: "Custom"},
			},
		}

		ensureStdResponses(comps)

		// Check that existing response is preserved
		if _, exists := comps.Responses["CustomResponse"]; !exists {
			t.Error("Expected existing CustomResponse to be preserved")
		}

		// Check that standard responses are added
		expectedResponses := []string{"BadRequest", "UnprocessableEntity", "InternalServerError"}
		for _, name := range expectedResponses {
			if _, exists := comps.Responses[name]; !exists {
				t.Errorf("Expected response '%s' to be added", name)
			}
		}
	})

	t.Run("do not overwrite existing standard responses", func(t *testing.T) {
		customBadRequest := &Response{Description: "Custom Bad Request"}
		comps := &Components{
			Responses: map[string]*Response{
				"BadRequest": customBadRequest,
			},
		}

		ensureStdResponses(comps)

		// Check that existing BadRequest is not overwritten
		if comps.Responses["BadRequest"] != customBadRequest {
			t.Error("Expected existing BadRequest response to not be overwritten")
		}
		if comps.Responses["BadRequest"].Description != "Custom Bad Request" {
			t.Error("Expected custom BadRequest description to be preserved")
		}

		// Check that other standard responses are still added
		if _, exists := comps.Responses["UnprocessableEntity"]; !exists {
			t.Error("Expected UnprocessableEntity to be added")
		}
		if _, exists := comps.Responses["InternalServerError"]; !exists {
			t.Error("Expected InternalServerError to be added")
		}
	})

	t.Run("verify response content structure", func(t *testing.T) {
		comps := &Components{}

		ensureStdResponses(comps)

		// Check BadRequest response structure
		badRequest := comps.Responses["BadRequest"]
		if badRequest.Description != "Bad Request - Validation failed" {
			t.Errorf("Expected BadRequest description to be 'Bad Request - Validation failed', got '%s'", badRequest.Description)
		}
		if badRequest.Content == nil {
			t.Fatal("Expected BadRequest to have Content")
		}
		if mediaType, exists := badRequest.Content["application/json"]; !exists {
			t.Error("Expected BadRequest to have application/json content")
		} else if mediaType.Schema == nil {
			t.Error("Expected BadRequest application/json to have Schema")
		} else if mediaType.Schema.Ref != "#/components/schemas/ValidationErrorResponse" {
			t.Errorf("Expected BadRequest schema ref to be ValidationErrorResponse, got '%s'", mediaType.Schema.Ref)
		}

		// Check UnprocessableEntity response structure
		unprocessable := comps.Responses["UnprocessableEntity"]
		if unprocessable.Description != "Unprocessable Entity - Request body could not be parsed" {
			t.Errorf("Expected UnprocessableEntity description, got '%s'", unprocessable.Description)
		}
		if mediaType, exists := unprocessable.Content["application/json"]; !exists {
			t.Error("Expected UnprocessableEntity to have application/json content")
		} else if mediaType.Schema.Ref != "#/components/schemas/ErrorResponse" {
			t.Errorf("Expected UnprocessableEntity schema ref to be ErrorResponse, got '%s'", mediaType.Schema.Ref)
		}

		// Check InternalServerError response structure
		internal := comps.Responses["InternalServerError"]
		if internal.Description != "Internal Server Error" {
			t.Errorf("Expected InternalServerError description, got '%s'", internal.Description)
		}
		if mediaType, exists := internal.Content["application/json"]; !exists {
			t.Error("Expected InternalServerError to have application/json content")
		} else if mediaType.Schema.Ref != "#/components/schemas/ErrorResponse" {
			t.Errorf("Expected InternalServerError schema ref to be ErrorResponse, got '%s'", mediaType.Schema.Ref)
		}
	})
}
