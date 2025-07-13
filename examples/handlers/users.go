package handlers

import (
	"context"

	"github.com/example/openapi-gen/pkg/api"
	"github.com/example/openapi-gen/pkg/unions"
)

// GetUserRequest represents the request body for getting a user
type GetUserRequest struct {
	// UserId is the ID of the user to retrieve
	UserId string `json:"userId"`
}

// CreateUserRequest represents the request body for creating a user
type UserResponse struct {
	// UserId is the ID of the user
	UserId string `json:"userId"`

	// Username is the username of the user
	Username string `json:"username"`
}

func GetUser(ctx context.Context, req *GetUserRequest) (*UserResponse, error) {
	// Handle getting user logic here
	return &UserResponse{UserId: req.UserId, Username: "example-user"}, nil
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	// Username is the username of the user to create
	Username string `json:"username"`
}

// CreateUser handles user creation requests
func CreateUser(ctx context.Context, req *CreateUserRequest) (*UserResponse, error) {
	// Handle user creation logic here
	return &UserResponse{UserId: "new-user-id", Username: req.Username}, nil
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	// UserId is the ID of the user to update
	UserId string `json:"userId"`

	// Username is the new username for the user
	Username string `json:"username"`
}

// UpdateUser handles user update requests
func UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UserResponse, error) {
	// Handle user update logic here
	return &UserResponse{UserId: req.UserId, Username: req.Username}, nil
}

// DeleteUserRequest represents the request body for deleting a user
type DeleteUserRequest struct {
	// UserId is the ID of the user to delete
	UserId string `json:"userId"`
}

// DeleteUser handles user deletion requests
func DeleteUser(ctx context.Context, req *DeleteUserRequest) (*api.NoContentResponse, error) {
	// Handle user deletion logic here
	return nil, nil
}

// ListUsersRequest represents the request body for listing users
type ListUsersRequest struct {
}

// AdminUserResponse represents an admin user with additional fields
type AdminUserResponse struct {
	UserResponse

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ListUsersResponse unions.Union2[[]AdminUserResponse, []UserResponse]

// ListUsers handles listing all users
func ListUsers(ctx context.Context, req *ListUsersRequest) (ListUsersResponse, error) {
	if ctx.Value("admin") != nil {
		// Return admin users if the context indicates admin access
		return ListUsersResponse{
			A: &[]AdminUserResponse{
				{UserResponse: UserResponse{UserId: "admin1", Username: "admin1"}, CreatedAt: "2023-01-01T00:00:00Z", UpdatedAt: "2023-01-02T00:00:00Z"},
				{UserResponse: UserResponse{UserId: "admin2", Username: "admin2"}, CreatedAt: "2023-01-03T00:00:00Z", UpdatedAt: "2023-01-04T00:00:00Z"},
			},
			B: nil,
		}, nil
	}
	// Return regular users if not admin

	return ListUsersResponse{
		A: nil,
		B: &[]UserResponse{
			{UserId: "user1", Username: "user1"},
			{UserId: "user2", Username: "user2"},
		},
	}, nil
}
