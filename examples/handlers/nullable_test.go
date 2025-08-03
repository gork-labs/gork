package handlers

import (
	"context"
)

// TestNullableRequest demonstrates nullable field handling
type TestNullableRequest struct {
	Body struct {
		RequiredField string  `json:"requiredField" validate:"required"`
		OptionalField *string `json:"optionalField,omitempty"`
		OptionalInt   *int    `json:"optionalInt,omitempty"`
	}
}

// TestNullableResponse demonstrates nullable field handling in responses
type TestNullableResponse struct {
	Body struct {
		ID            int     `json:"id"`
		Name          string  `json:"name"`
		OptionalValue *string `json:"optionalValue,omitempty"`
	}
}

// NullableHandler handles nullable field testing
func NullableHandler(_ context.Context, req *TestNullableRequest) (*TestNullableResponse, error) {
	var optionalValue *string
	if req.Body.OptionalField != nil {
		optionalValue = req.Body.OptionalField
	}

	return &TestNullableResponse{
		Body: struct {
			ID            int     `json:"id"`
			Name          string  `json:"name"`
			OptionalValue *string `json:"optionalValue,omitempty"`
		}{
			ID:            1,
			Name:          req.Body.RequiredField,
			OptionalValue: optionalValue,
		},
	}, nil
}
