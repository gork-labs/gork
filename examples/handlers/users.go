package handlers

import (
	"context"
	"fmt"

	"github.com/gork-labs/gork/pkg/rules"
	"github.com/gork-labs/gork/pkg/unions"
)

// GetUserRequest represents the request for getting a user.
type GetUserRequest struct {
	Path struct {
		// UserID is the ID of the user to retrieve
		UserID string `gork:"userId" validate:"required"`
	}
}

// UserType represents user data structure.
type UserType struct {
	// UserID is the ID of the user
	UserID string `gork:"userID"`

	// Username is the username of the user
	Username string `gork:"username"`
}

// UserResponse represents the response for user operations.
type UserResponse struct {
	Body UserType
}

// GetUser handles user retrieval requests.
func GetUser(_ context.Context, req GetUserRequest) (*UserResponse, error) {
	// Handle getting user logic here
	return &UserResponse{
		Body: UserType{
			UserID:   req.Path.UserID,
			Username: "example-user",
		},
	}, nil
}

// CreateUserRequest represents the request body for creating a user.
type CreateUserRequest struct {
	Path struct {
		// No path parameters for this endpoint
	}
	Query struct {
		// ReturnTo specifies where to redirect after user creation
		ReturnTo string `gork:"return-to"`
	}
	Body struct {
		// Username is the username of the user to create
		Username string `gork:"username" validate:"required"`
	}
}

// CreateUser handles user creation requests.
func CreateUser(_ context.Context, req CreateUserRequest) (*UserResponse, error) {
	// Handle user creation logic here
	return &UserResponse{
		Body: UserType{
			UserID:   "new-user-id",
			Username: req.Body.Username,
		},
	}, nil
}

// UpdateUserRequest represents the request body for updating a user.
type UpdateUserRequest struct {
	Path struct {
		// UserID is the ID of the user to update
		UserID string `gork:"userId" validate:"required"`
	}
	Headers struct {
		// Version specifies the user version for concurrency control
		Version int `gork:"X-User-Version"`
	}
	Body struct {
		// UserID is the ID of the user to update
		UserID string `gork:"userID" validate:"required"`
		// Username is the new username for the user
		Username string `gork:"username" validate:"required"`
	}
}

// UpdateUser handles user update requests.
func UpdateUser(_ context.Context, req UpdateUserRequest) (*UserResponse, error) {
	// Handle user update logic here
	return &UserResponse{
		Body: UserType{
			UserID:   req.Body.UserID,
			Username: req.Body.Username,
		},
	}, nil
}

// DeleteUserRequest represents the request body for deleting a user.
type DeleteUserRequest struct {
	Path struct {
		// UserID is the ID of the user to delete
		UserID string `gork:"userId" validate:"required"`
	}
	Query struct {
		// Force specifies whether to force deletion
		Force bool `gork:"force"`
	}
	Body struct {
		// UserID is the ID of the user to delete
		UserID string `gork:"userID" validate:"required"`
	}
}

// DeleteUser handles user deletion requests.
func DeleteUser(_ context.Context, _ DeleteUserRequest) (*struct{}, error) {
	// Handle user deletion logic here
	return nil, nil
}

// ListUsersRequest represents the request body for listing users.
type ListUsersRequest struct {
	Query struct {
		// Limit is the maximum number of users to return
		Limit int `gork:"limit"`

		// Offset is the number of users to skip
		Offset int `gork:"offset"`
	}
}

// AdminUserType represents admin user data structure.
type AdminUserType struct {
	// UserID is the ID of the user
	UserID string `gork:"userID"`

	// Username is the username of the user
	Username string `gork:"username"`

	// CreatedAt is when the user was created
	CreatedAt string `gork:"createdAt"`

	// UpdatedAt is when the user was last updated
	UpdatedAt string `gork:"updatedAt"`
}

// AdminUserResponse represents an admin user response with additional fields.
type AdminUserResponse struct {
	Body AdminUserType
}

// ListUsersResponse represents the response for listing users.
type ListUsersResponse struct {
	Body unions.Union2[[]AdminUserType, []UserType]
}

// ListUsers handles listing all users.
func ListUsers(ctx context.Context, _ ListUsersRequest) (ListUsersResponse, error) {
	if ctx.Value("admin") != nil {
		// Return admin users if the context indicates admin access
		return ListUsersResponse{
			Body: unions.Union2[[]AdminUserType, []UserType]{
				A: &[]AdminUserType{
					{
						UserID:    "admin1",
						Username:  "admin1",
						CreatedAt: "2023-01-01T00:00:00Z",
						UpdatedAt: "2023-01-02T00:00:00Z",
					},
					{
						UserID:    "admin2",
						Username:  "admin2",
						CreatedAt: "2023-01-03T00:00:00Z",
						UpdatedAt: "2023-01-04T00:00:00Z",
					},
				},
				B: nil,
			},
		}, nil
	}
	// Return regular users if not admin

	return ListUsersResponse{
		Body: unions.Union2[[]AdminUserType, []UserType]{
			A: nil,
			B: &[]UserType{
				{
					UserID:   "user1",
					Username: "user1",
				},
				{
					UserID:   "user2",
					Username: "user2",
				},
			},
		},
	}, nil
}

// Simulated ownership map for demo purposes.
var exampleItemOwners = map[string]string{
	"123": "alice",
}

// Register a simple owned_by rule for demonstration when the package is loaded.
func init() {
	rules.Register("owned_by", func(_ context.Context, itemID *string, currentUser string) (bool, error) {
		if itemID == nil {
			return false, fmt.Errorf("item id is nil")
		}
		owner := exampleItemOwners[*itemID]
		if owner != currentUser {
			return false, fmt.Errorf("item %s is not owned by %s", *itemID, currentUser)
		}
		return true, nil
	})
}

// UpdateOwnedItemRequest is a demo request guarded by a rule expression.
type UpdateOwnedItemRequest struct {
	Path struct {
		ItemID string `gork:"itemId" rule:"owned_by($current_user)"`
	}
}

// UpdateOwnedItemResponse is a demo empty response.
type UpdateOwnedItemResponse struct {
	Body struct{} `gork:"body"`
}

// UpdateOwnedItem is a demo handler that runs after validation and rules.
func UpdateOwnedItem(_ context.Context, _ UpdateOwnedItemRequest) (*UpdateOwnedItemResponse, error) {
	// Validation (including rules) is applied by the router before this handler runs.
	return &UpdateOwnedItemResponse{}, nil
}
