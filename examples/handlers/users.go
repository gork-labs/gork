package handlers

import (
	"context"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/gork-labs/gork/pkg/unions"
)

// GetUserRequest represents the request body for getting a user.
type GetUserRequest struct {
	// UserID is the ID of the user to retrieve
	UserID string `json:"userID"`
}

// UserResponse represents the response for user operations.
type UserResponse struct {
	// UserID is the ID of the user
	UserID string `json:"userID"`

	// Username is the username of the user
	Username string `json:"username"`
}

// GetUser handles user retrieval requests.
func GetUser(_ context.Context, req *GetUserRequest) (*UserResponse, error) {
	// Handle getting user logic here
	return &UserResponse{UserID: req.UserID, Username: "example-user"}, nil
}

// CreateUserRequest represents the request body for creating a user.
type CreateUserRequest struct {
	// Username is the username of the user to create
	Username string `json:"username"`

	ReturnTo string `openapi:"return-to,in=query"`
}

// CreateUser handles user creation requests.
func CreateUser(_ context.Context, req *CreateUserRequest) (*UserResponse, error) {
	// Handle user creation logic here
	return &UserResponse{UserID: "new-user-id", Username: req.Username}, nil
}

// UpdateUserRequest represents the request body for updating a user.
type UpdateUserRequest struct {
	// UserID is the ID of the user to update
	UserID string `json:"userID"`

	// Username is the new username for the user
	Username string `json:"username"`

	Version int `openapi:"X-User-Version,in=header"`
}

// UpdateUser handles user update requests.
func UpdateUser(_ context.Context, req *UpdateUserRequest) (*UserResponse, error) {
	// Handle user update logic here
	return &UserResponse{UserID: req.UserID, Username: req.Username}, nil
}

// DeleteUserRequest represents the request body for deleting a user.
type DeleteUserRequest struct {
	// UserID is the ID of the user to delete
	UserID string `json:"userID"`

	Force bool `openapi:"force,in=query"`
}

// DeleteUser handles user deletion requests.
func DeleteUser(_ context.Context, _ *DeleteUserRequest) (*api.NoContentResponse, error) {
	// Handle user deletion logic here
	return nil, nil
}

// ListUsersRequest represents the request body for listing users.
type ListUsersRequest struct{}

// AdminUserResponse represents an admin user with additional fields.
type AdminUserResponse struct {
	UserResponse

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ListUsersResponse represents the response for listing users.
type ListUsersResponse unions.Union2[[]AdminUserResponse, []UserResponse]

// ListUsers handles listing all users.
func ListUsers(ctx context.Context, _ *ListUsersRequest) (ListUsersResponse, error) {
	if ctx.Value("admin") != nil {
		// Return admin users if the context indicates admin access
		return ListUsersResponse{
			A: &[]AdminUserResponse{
				{UserResponse: UserResponse{UserID: "admin1", Username: "admin1"}, CreatedAt: "2023-01-01T00:00:00Z", UpdatedAt: "2023-01-02T00:00:00Z"},
				{UserResponse: UserResponse{UserID: "admin2", Username: "admin2"}, CreatedAt: "2023-01-03T00:00:00Z", UpdatedAt: "2023-01-04T00:00:00Z"},
			},
			B: nil,
		}, nil
	}
	// Return regular users if not admin

	return ListUsersResponse{
		A: nil,
		B: &[]UserResponse{
			{UserID: "user1", Username: "user1"},
			{UserID: "user2", Username: "user2"},
		},
	}, nil
}
