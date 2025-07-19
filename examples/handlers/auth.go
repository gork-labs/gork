// Package handlers contains HTTP handler functions for the example API.
package handlers

import "context"

// LoginRequest represents the request body for the login endpoint.
type LoginRequest struct {
	// Username is the user's username
	Username string `json:"username"`

	// Password is the user's password
	Password string `json:"password"`
}

// LoginResponse represents the response body for the login endpoint.
type LoginResponse struct {
	// Token is the JWT token for the authenticated user
	Token string `json:"token"`
}

// Login handles user login requests.
func Login(_ context.Context, _ *LoginRequest) (*LoginResponse, error) {
	// Handle login logic here
	return &LoginResponse{Token: "example-token"}, nil
}
