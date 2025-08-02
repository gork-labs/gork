package handlers

import (
	"context"
)

// TestNullableRequest demonstrates nullable field handling
type TestNullableRequest struct {
	RequiredField string  `json:"requiredField" validate:"required"`
	OptionalField *string `json:"optionalField,omitempty"`
	OptionalInt   *int    `json:"optionalInt,omitempty"`
}

// TestNullableResponse demonstrates nullable field handling in responses
type TestNullableResponse struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	OptionalValue *string `json:"optionalValue,omitempty"`
}

// NullableHandler handles nullable field testing
func NullableHandler(_ context.Context, req *TestNullableRequest) (*TestNullableResponse, error) {
	var optionalValue *string
	if req.OptionalField != nil {
		optionalValue = req.OptionalField
	}

	return &TestNullableResponse{
		ID:            1,
		Name:          req.RequiredField,
		OptionalValue: optionalValue,
	}, nil
}
