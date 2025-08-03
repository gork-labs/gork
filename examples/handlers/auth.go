// Package handlers contains HTTP handler functions for the example API.
package handlers

import "context"

// LoginRequest represents the request body for the login endpoint.
type LoginRequest struct {
	Body struct {
		// Username is the user's username
		Username string `gork:"username" validate:"required"`

		// Password is the user's password
		Password string `gork:"password" validate:"required"`
	}
}

// LoginResponse represents the response body for the login endpoint.
type LoginResponse struct {
	Body struct {
		// Token is the JWT token for the authenticated user
		Token string `gork:"token"`
	}
}

// Login handles user login requests.
func Login(_ context.Context, _ *LoginRequest) (*LoginResponse, error) {
	// Handle login logic here
	return &LoginResponse{
		Body: struct {
			Token string `gork:"token"`
		}{
			Token: "example-token",
		},
	}, nil
}
