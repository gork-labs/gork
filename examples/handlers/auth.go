// Package handlers contains HTTP handler functions for the example API.
package handlers

import (
	"context"
	"net/http"

	"github.com/gork-labs/gork/pkg/api"
)

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
func Login(_ context.Context, req LoginRequest) (*LoginResponse, error) {
	if req.Body.Password != "example-password" {
		return nil, api.NewHTTPError(http.StatusUnauthorized, "The username or the password is wrong.")
	}
	return &LoginResponse{
		Body: struct {
			Token string `gork:"token"`
		}{
			Token: "example-token",
		},
	}, nil
}

// OAuthCallbackRequest represents the query of the page where the identity provider sends the browser.
type OAuthCallbackRequest struct {
	Query struct {
		// Code is the authorization code from the identity provider
		Code string `gork:"code" validate:"required"`

		// State is the value that the login page gave to the identity provider
		State string `gork:"state" validate:"required"`
	}
}

// OAuthCallbackResponse sends the browser to the next page.
type OAuthCallbackResponse struct {
	Headers struct {
		// Location is the page that the browser opens next
		Location string `gork:"Location"`
	}
}

// OAuthCallback completes a login with an identity provider and sends the browser to the home page.
func OAuthCallback(_ context.Context, req OAuthCallbackRequest) (*OAuthCallbackResponse, error) {
	if req.Query.State != "example-state" {
		return nil, api.NewHTTPError(http.StatusForbidden, "The login state is not valid.")
	}
	resp := &OAuthCallbackResponse{}
	resp.Headers.Location = "/"
	return resp, nil
}
