package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/gork-labs/gork/pkg/gorkson"
)

// User represents a user entity
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserService simulates a user service
type UserService interface {
	GetByID(ctx context.Context, id int) (*User, error)
}

// mockUserService is a simple mock implementation
type mockUserService struct {
	users map[int]User
}

func (s *mockUserService) GetByID(ctx context.Context, id int) (*User, error) {
	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found: %d", id)
	}
	return &user, nil
}

// UserCodec handles entity resolution - parsing user IDs to User structs
type UserCodec struct {
	userService UserService
}

func NewUserCodec(userService UserService) UserCodec {
	return UserCodec{userService: userService}
}

func (c UserCodec) Parse(ctx context.Context, value string) (*User, error) {
	// Parse the ID from string
	userID, err := strconv.Atoi(value)
	if err != nil {
		return nil, api.NewInvalidFormatError("User", value, fmt.Sprintf("invalid user ID format: %v", err))
	}

	// Fetch the full user from database/service
	user, err := c.userService.GetByID(ctx, userID)
	if err != nil {
		return nil, gorkson.NewParseError("User", value, err)
	}

	return user, nil
}

func (c UserCodec) Format(ctx context.Context, value *User) (string, error) {
	if value == nil {
		return "", gorkson.NewFormatError("User", "nil user")
	}

	// Format the user back to just its ID for transport
	return strconv.Itoa(value.ID), nil
}

func (c UserCodec) Schema() api.OpenAPISchema {
	return api.OpenAPISchema{
		Type:        api.OpenAPITypeInteger,
		Format:      "int32",
		Example:     123,
		Description: "User ID that will be resolved to full user entity",
		Minimum:     ptr(1.0),
	}
}

// Priority is a custom enum type using iota for internal representation.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

// String returns the string representation of Priority.
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// ParsePriority converts string to Priority enum.
func ParsePriority(s string) (Priority, error) {
	switch s {
	case "low":
		return PriorityLow, nil
	case "medium":
		return PriorityMedium, nil
	case "high":
		return PriorityHigh, nil
	default:
		return PriorityMedium, fmt.Errorf("invalid priority: %s", s)
	}
}

// PriorityCodec handles enum validation and conversion
type PriorityCodec struct{}

func (c PriorityCodec) Parse(ctx context.Context, value string) (*Priority, error) {
	priority, err := ParsePriority(value)
	if err != nil {
		return nil, api.NewInvalidFormatError("Priority", value, "must be one of: low, medium, high")
	}
	return &priority, nil
}

func (c PriorityCodec) Format(ctx context.Context, value *Priority) (string, error) {
	if value == nil {
		defaultPriority := PriorityMedium
		return defaultPriority.String(), nil
	}
	return value.String(), nil
}

func (c PriorityCodec) Schema() api.OpenAPISchema {
	return api.OpenAPISchema{
		Type:        api.OpenAPITypeString,
		Enum:        []interface{}{"low", "medium", "high"},
		Example:     "medium",
		Description: "Task priority level (internal iota, external string)",
	}
}

// Request/Response structs demonstrating codec usage
type CreateTaskRequest struct {
	Path struct {
		ProjectID int `gork:"projectId"`
	}
	Body struct {
		Title       string    `json:"title" gork:"title"`
		Description string    `json:"description" gork:"description"`
		Priority    Priority  `json:"priority" gork:"priority"`
		AssigneeID  User      `json:"assignee" gork:"assignee"` // Resolved from ID
		DueDate     time.Time `json:"dueDate" gork:"dueDate"`   // Parsed from RFC3339
		Completed   bool      `json:"completed" gork:"completed"`
	}
}

type CreateTaskResponse struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    Priority  `json:"priority"` // Formatted back to string
	Assignee    User      `json:"assignee"` // Full user object in response
	DueDate     time.Time `json:"dueDate"`  // Formatted back to RFC3339
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SearchTasksRequest struct {
	Query struct {
		Priority     Priority  `gork:"priority"`      // Optional priority filter
		AssigneeID   User      `gork:"assignee"`      // Resolved from ID
		CreatedAfter time.Time `gork:"created_after"` // Time parsing
		Completed    bool      `gork:"completed"`     // Boolean parsing
		Limit        int       `gork:"limit"`         // Integer parsing
	}
}

type SearchTasksResponse struct {
	Tasks []CreateTaskResponse `json:"tasks"`
	Total int                  `json:"total"`
	Query SearchTasksRequest   `json:"query"` // Echo back query params (all formatted)
}

// Handler functions demonstrating automatic type conversion
func CreateTask(ctx context.Context, req CreateTaskRequest) (*CreateTaskResponse, error) {
	fmt.Printf("Creating task in project %d\n", req.Path.ProjectID)
	fmt.Printf("Assigned to user: %+v\n", req.Body.AssigneeID)
	fmt.Printf("Due date: %s\n", req.Body.DueDate.Format(time.RFC3339))
	fmt.Printf("Priority: %s\n", req.Body.Priority)
	fmt.Printf("Completed: %t\n", req.Body.Completed)

	// All fields are automatically parsed by their respective codecs
	return &CreateTaskResponse{
		ID:          1,
		Title:       req.Body.Title,
		Description: req.Body.Description,
		Priority:    req.Body.Priority,
		Assignee:    req.Body.AssigneeID,
		DueDate:     req.Body.DueDate,
		Completed:   req.Body.Completed,
		CreatedAt:   time.Now(),
	}, nil
}

func SearchTasks(ctx context.Context, req SearchTasksRequest) (*SearchTasksResponse, error) {
	fmt.Printf("Searching tasks with priority: %s\n", req.Query.Priority)
	fmt.Printf("Assigned to user: %+v\n", req.Query.AssigneeID)
	fmt.Printf("Created after: %s\n", req.Query.CreatedAfter.Format(time.RFC3339))
	fmt.Printf("Completed: %t\n", req.Query.Completed)
	fmt.Printf("Limit: %d\n", req.Query.Limit)

	// Mock response
	tasks := []CreateTaskResponse{
		{
			ID:          1,
			Title:       "Example Task",
			Description: "This is an example task",
			Priority:    req.Query.Priority,
			Assignee:    req.Query.AssigneeID,
			DueDate:     req.Query.CreatedAfter.Add(24 * time.Hour),
			Completed:   req.Query.Completed,
			CreatedAt:   time.Now(),
		},
	}

	return &SearchTasksResponse{
		Tasks: tasks,
		Total: len(tasks),
		Query: req, // All fields will be formatted back to transportable strings
	}, nil
}

// Demo function showing codec setup and usage
func main() {
	fmt.Println("=== Type Codec System Demo ===\n")

	// Create mock user service
	userService := &mockUserService{
		users: map[int]User{
			123: {ID: 123, Name: "John Doe", Email: "john@example.com"},
			456: {ID: 456, Name: "Jane Smith", Email: "jane@example.com"},
		},
	}

	// Register all codecs
	fmt.Println("1. Registering type codecs...")
	gorkson.RegisterCodec[time.Time](gorkson.TimeCodec{})
	gorkson.RegisterCodec[User](NewUserCodec(userService))
	gorkson.RegisterCodec[Priority](PriorityCodec{})

	// Demonstrate parsing capabilities
	fmt.Println("\n2. Demonstrating codec parsing...")

	parser := api.NewConventionParser()
	ctx := context.Background()

	// Test time formatting for JSON responses (preserves native type)
	timeJSONValue, err := parser.FormatFieldValue(ctx, ptr(time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)))
	if err != nil {
		fmt.Printf("Time JSON formatting error: %v\n", err)
	} else {
		fmt.Printf("Time JSON value: %v (type: %T)\n", timeJSONValue, timeJSONValue)
	}

	// Test user resolution (this would normally happen during request parsing)
	userResult, err := userService.GetByID(ctx, 123)
	if err != nil {
		fmt.Printf("User lookup error: %v\n", err)
	} else {
		fmt.Printf("User resolved: %+v\n", *userResult)

		// Test user formatting for JSON responses (preserves native types)
		userJSONValue, err := parser.FormatFieldValue(ctx, userResult)
		if err != nil {
			fmt.Printf("User JSON formatting error: %v\n", err)
		} else {
			fmt.Printf("User JSON value: %v (type: %T)\n", userJSONValue, userJSONValue)
		}
	}

	// Test priority parsing
	priorityCodec := PriorityCodec{}
	priority, err := priorityCodec.Parse(ctx, "high")
	if err != nil {
		fmt.Printf("Priority parsing error: %v\n", err)
	} else {
		fmt.Printf("Priority parsed: %s\n", *priority)
	}

	// Test invalid priority
	_, err = priorityCodec.Parse(ctx, "invalid")
	if err != nil {
		fmt.Printf("Invalid priority error: %v\n", err)
	}

	fmt.Println("\n3. Schema generation example...")

	// Show schemas that would be generated for OpenAPI
	schemas := map[string]gorkson.OpenAPISchema{
		"User":      NewUserCodec(userService).Schema(),
		"Priority":  PriorityCodec{}.Schema(),
		"time.Time": gorkson.TimeCodec{}.Schema(),
	}

	for typeName, schema := range schemas {
		schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
		fmt.Printf("%s schema:\n%s\n\n", typeName, schemaJSON)
	}

	fmt.Println("4. End-to-end request/response handling...")

	// Simulate request handling (in real usage, these would be parsed from HTTP parameters)
	createReq := CreateTaskRequest{}
	createReq.Path.ProjectID = 1
	createReq.Body.Title = "Test Task"
	createReq.Body.Description = "This is a test task"
	createReq.Body.Priority = PriorityHigh
	createReq.Body.AssigneeID = User{ID: 123, Name: "John Doe", Email: "john@example.com"}
	createReq.Body.DueDate = time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
	createReq.Body.Completed = false

	response, err := CreateTask(ctx, createReq)
	if err != nil {
		fmt.Printf("Error creating task: %v\n", err)
	} else {
		responseJSON, _ := json.MarshalIndent(response, "", "  ")
		fmt.Printf("Task created successfully:\n%s\n", responseJSON)
	}

	fmt.Println("\n=== Demo Complete ===")
}

// Helper function to create pointers
func ptr[T any](v T) *T {
	return &v
}
