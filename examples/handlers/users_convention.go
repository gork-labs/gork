package handlers

import (
	"context"
	"time"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/gork-labs/gork/pkg/unions"
)

// Convention Over Configuration Examples

// GetUserConventionRequest demonstrates the new convention structure.
type GetUserConventionRequest struct {
	Path struct {
		UserID string `gork:"user_id" validate:"required,uuid"`
	}
	Query struct {
		IncludeProfile bool     `gork:"include_profile"`
		Fields         []string `gork:"fields"`
	}
}

// GetUserConventionResponse demonstrates response structure.
type GetUserConventionResponse struct {
	Body struct {
		ID      string    `gork:"id"`
		Name    string    `gork:"name"`
		Email   string    `gork:"email"`
		Created time.Time `gork:"created"`
		Profile struct {
			Bio  string   `gork:"bio"`
			Tags []string `gork:"tags"`
		} `gork:"profile"`
	}
}

// GetUserConvention handles user retrieval using Convention Over Configuration.
func GetUserConvention(_ context.Context, req GetUserConventionRequest) (*GetUserConventionResponse, error) {
	return &GetUserConventionResponse{
		Body: struct {
			ID      string    `gork:"id"`
			Name    string    `gork:"name"`
			Email   string    `gork:"email"`
			Created time.Time `gork:"created"`
			Profile struct {
				Bio  string   `gork:"bio"`
				Tags []string `gork:"tags"`
			} `gork:"profile"`
		}{
			ID:      req.Path.UserID,
			Name:    "John Doe",
			Email:   "john@example.com",
			Created: time.Now(),
			Profile: struct {
				Bio  string   `gork:"bio"`
				Tags []string `gork:"tags"`
			}{
				Bio:  "Software developer",
				Tags: req.Query.Fields,
			},
		},
	}, nil
}

// CreateUserConventionRequest demonstrates body and headers.
type CreateUserConventionRequest struct {
	Body struct {
		Name     string                 `gork:"name" validate:"required,min=1,max=100"`
		Email    string                 `gork:"email" validate:"required,email"`
		Age      int                    `gork:"age" validate:"min=18,max=120"`
		Metadata map[string]interface{} `gork:"metadata"`
	}
	Headers struct {
		Authorization string `gork:"Authorization" validate:"required"`
		ContentType   string `gork:"Content-Type"`
	}
}

// CreateUserConventionResponse demonstrates response with headers.
type CreateUserConventionResponse struct {
	Body struct {
		ID      string    `gork:"id"`
		Name    string    `gork:"name"`
		Email   string    `gork:"email"`
		Created time.Time `gork:"created"`
	}
	Headers struct {
		Location     string `gork:"Location"`
		CacheControl string `gork:"Cache-Control"`
	}
}

// CreateUserConvention handles user creation using Convention Over Configuration.
func CreateUserConvention(_ context.Context, req CreateUserConventionRequest) (*CreateUserConventionResponse, error) {
	userID := "user-123"

	return &CreateUserConventionResponse{
		Body: struct {
			ID      string    `gork:"id"`
			Name    string    `gork:"name"`
			Email   string    `gork:"email"`
			Created time.Time `gork:"created"`
		}{
			ID:      userID,
			Name:    req.Body.Name,
			Email:   req.Body.Email,
			Created: time.Now(),
		},
		Headers: struct {
			Location     string `gork:"Location"`
			CacheControl string `gork:"Cache-Control"`
		}{
			Location:     "/users/" + userID,
			CacheControl: "no-cache",
		},
	}, nil
}

// Union Types with Convention Over Configuration

// EmailLogin represents email-based login.
type EmailLogin struct {
	Type     string `gork:"type,discriminator=email" validate:"required"`
	Email    string `gork:"email" validate:"required,email"`
	Password string `gork:"password" validate:"required"`
}

// PhoneLogin represents phone-based login.
type PhoneLogin struct {
	Type  string `gork:"type,discriminator=phone" validate:"required"`
	Phone string `gork:"phone" validate:"required,e164"`
	Code  string `gork:"code" validate:"required,len=6"`
}

// OAuthLogin represents OAuth-based login.
type OAuthLogin struct {
	Type        string `gork:"type,discriminator=oauth" validate:"required"`
	Provider    string `gork:"provider" validate:"required,oneof=google facebook github"`
	AccessToken string `gork:"access_token" validate:"required"`
}

// LoginConventionRequest demonstrates union types in body.
type LoginConventionRequest struct {
	Body struct {
		LoginMethod unions.Union3[EmailLogin, PhoneLogin, OAuthLogin] `gork:"login_method" validate:"required"`
		RememberMe  bool                                              `gork:"remember_me"`
	}
}

// LoginConventionResponse demonstrates successful login response.
type LoginConventionResponse struct {
	Body struct {
		Token     string    `gork:"token"`
		ExpiresAt time.Time `gork:"expires_at"`
		User      struct {
			ID   string `gork:"id"`
			Name string `gork:"name"`
		} `gork:"user"`
	}
	Headers struct {
		Location     string `gork:"Location"`
		CacheControl string `gork:"Cache-Control"`
	}
	Cookies struct {
		SessionToken string `gork:"session_token"`
		Preferences  string `gork:"preferences"`
	}
}

// LoginConvention handles login using union types and Convention Over Configuration.
func LoginConvention(_ context.Context, req LoginConventionRequest) (*LoginConventionResponse, error) {
	// Access the union type using accessor methods (when implemented)
	loginMethod := req.Body.LoginMethod

	// For now, just create a successful response
	// In a real implementation, you would handle different login methods
	_ = loginMethod

	return &LoginConventionResponse{
		Body: struct {
			Token     string    `gork:"token"`
			ExpiresAt time.Time `gork:"expires_at"`
			User      struct {
				ID   string `gork:"id"`
				Name string `gork:"name"`
			} `gork:"user"`
		}{
			Token:     "jwt-token-here",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			User: struct {
				ID   string `gork:"id"`
				Name string `gork:"name"`
			}{
				ID:   "user-123",
				Name: "John Doe",
			},
		},
		Headers: struct {
			Location     string `gork:"Location"`
			CacheControl string `gork:"Cache-Control"`
		}{
			Location:     "/dashboard",
			CacheControl: "no-cache",
		},
		Cookies: struct {
			SessionToken string `gork:"session_token"`
			Preferences  string `gork:"preferences"`
		}{
			SessionToken: "session-123",
			Preferences:  "dark-mode",
		},
	}, nil
}

// Complex Request with All Sections

// UpdateUserConventionRequest demonstrates all sections and custom validation.
type UpdateUserConventionRequest struct {
	Path struct {
		UserID string `gork:"user_id" validate:"required,uuid"`
	}
	Query struct {
		Force  bool `gork:"force"`
		Notify bool `gork:"notify"`
	}
	Body struct {
		Name    string  `gork:"name" validate:"min=1,max=100"`
		Email   *string `gork:"email" validate:"omitempty,email"`
		Profile struct {
			Bio  string   `gork:"bio" validate:"max=500"`
			Tags []string `gork:"tags" validate:"dive,min=1,max=20"`
		} `gork:"profile"`
	}
	Headers struct {
		Authorization string `gork:"Authorization" validate:"required"`
		IfMatch       string `gork:"If-Match"`
	}
	Cookies struct {
		SessionID   string `gork:"session_id"`
		Preferences string `gork:"preferences"`
	}
}

// Validate implements custom request-level validation.
func (r *UpdateUserConventionRequest) Validate() error {
	// Example cross-section validation
	if r.Query.Force && r.Body.Name == "" {
		return &api.RequestValidationError{
			Errors: []string{"name is required when force flag is set"},
		}
	}

	// Business rule: notification requires email
	if r.Query.Notify && (r.Body.Email == nil || *r.Body.Email == "") {
		return &api.RequestValidationError{
			Errors: []string{"email is required when notification is enabled"},
		}
	}

	return nil
}

// UpdateUserConventionResponse demonstrates comprehensive response.
type UpdateUserConventionResponse struct {
	Body struct {
		ID      string    `gork:"id"`
		Name    string    `gork:"name"`
		Email   string    `gork:"email"`
		Updated time.Time `gork:"updated"`
		Profile struct {
			Bio  string   `gork:"bio"`
			Tags []string `gork:"tags"`
		} `gork:"profile"`
	}
	Headers struct {
		Location string `gork:"Location"`
		ETag     string `gork:"ETag"`
	}
}

// UpdateUserConvention handles comprehensive user updates.
func UpdateUserConvention(_ context.Context, req UpdateUserConventionRequest) (*UpdateUserConventionResponse, error) {
	return &UpdateUserConventionResponse{
		Body: struct {
			ID      string    `gork:"id"`
			Name    string    `gork:"name"`
			Email   string    `gork:"email"`
			Updated time.Time `gork:"updated"`
			Profile struct {
				Bio  string   `gork:"bio"`
				Tags []string `gork:"tags"`
			} `gork:"profile"`
		}{
			ID:   req.Path.UserID,
			Name: req.Body.Name,
			Email: func() string {
				if req.Body.Email != nil {
					return *req.Body.Email
				}
				return ""
			}(),
			Updated: time.Now(),
			Profile: struct {
				Bio  string   `gork:"bio"`
				Tags []string `gork:"tags"`
			}{
				Bio:  req.Body.Profile.Bio,
				Tags: req.Body.Profile.Tags,
			},
		},
		Headers: struct {
			Location string `gork:"Location"`
			ETag     string `gork:"ETag"`
		}{
			Location: "/users/" + req.Path.UserID,
			ETag:     "\"abc123\"",
		},
	}, nil
}
