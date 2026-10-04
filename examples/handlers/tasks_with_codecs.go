package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gork-labs/gork/pkg/gorkson"
)

// Priority represents task priority levels using iota for internal representation.
// This provides type safety, efficient comparison, and easy ordering while
// maintaining clean string representation for APIs.
type Priority int

const (
	// PriorityLow represents the lowest priority level (0).
	PriorityLow Priority = iota
	// PriorityMedium represents the medium priority level (1).
	PriorityMedium
	// PriorityHigh represents the highest priority level (2).
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

// IsValid returns true if the priority is a valid enum value.
func (p Priority) IsValid() bool {
	return p >= PriorityLow && p <= PriorityHigh
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
		return PriorityMedium, fmt.Errorf("invalid priority: %s, valid values: low, medium, high", s)
	}
}

// PriorityCodec handles enum validation and conversion for Priority type.
type PriorityCodec struct{}

// Parse validates and converts string to Priority enum using iota values.
// Enum validation is automatically handled by schema validation.
func (c PriorityCodec) Parse(_ context.Context, value string) (*Priority, error) {
	priority, err := ParsePriority(value)
	if err != nil {
		return nil, gorkson.NewParseError("Priority", value, err)
	}
	return &priority, nil
}

// Format converts Priority enum (iota) back to string representation.
func (c PriorityCodec) Format(_ context.Context, value *Priority) (string, error) {
	if value == nil {
		defaultPriority := PriorityMedium
		return defaultPriority.String(), nil
	}
	return value.String(), nil
}

// Schema returns OpenAPI schema for Priority type.
func (c PriorityCodec) Schema() gorkson.OpenAPISchema {
	return gorkson.OpenAPISchema{
		Type:        gorkson.OpenAPITypeString,
		Enum:        []interface{}{"low", "medium", "high"},
		Example:     "medium",
		Description: "Task priority level (internal iota, external string)",
	}
}

// Task represents a task entity with full information.
type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// TaskCodec demonstrates entity resolution - converting ID strings to full task entities.
type TaskCodec struct{}

// Parse converts task ID string to Task struct with resolved info.
// Numeric range validation (minimum: 1) is automatically handled by schema validation.
func (c TaskCodec) Parse(_ context.Context, value string) (*Task, error) {
	id, err := strconv.Atoi(value)
	if err != nil {
		return nil, gorkson.NewParseError("Task", value, err)
	}

	// Simulate fetching task info (in real app, this would be a database lookup)
	taskTitles := map[int]string{
		1: "Setup Development Environment",
		2: "Implement User Authentication",
		3: "Write API Documentation",
	}

	title, exists := taskTitles[id]
	if !exists {
		return nil, gorkson.NewParseError("Task", value, fmt.Errorf("task not found"))
	}

	return &Task{ID: id, Title: title}, nil
}

// Format converts Task back to just the ID string for transport.
func (c TaskCodec) Format(_ context.Context, value *Task) (string, error) {
	if value == nil {
		return "", gorkson.NewFormatError("Task", "nil task")
	}
	return strconv.Itoa(value.ID), nil
}

// Schema returns OpenAPI schema for Task (transported as integer ID).
func (c TaskCodec) Schema() gorkson.OpenAPISchema {
	return gorkson.OpenAPISchema{
		Type:        gorkson.OpenAPITypeInteger,
		Format:      "int32",
		Example:     1,
		Description: "Task ID that resolves to full task information",
		Minimum:     ptr(1.0),
	}
}

// GetTaskRequest demonstrates codec usage in path parameters.
type GetTaskRequest struct {
	Path struct {
		Task        Task      `gork:"taskId"`      // Uses TaskCodec for entity resolution
		CompletedAt time.Time `gork:"completedAt"` // Uses TimeCodec for RFC3339 parsing
	}
	Query struct {
		Priority Priority `gork:"priority"` // Uses PriorityCodec for enum validation
	}
}

// GetTaskResponse demonstrates codec usage in responses.
type GetTaskResponse struct {
	Body struct {
		Task        Task      `gork:"task"`        // Will be formatted back to ID
		Priority    Priority  `gork:"priority"`    // Will be formatted back to string
		CompletedAt time.Time `gork:"completedAt"` // Will be formatted back to RFC3339
		Message     string    `gork:"message"`
	}
}

// GetTask demonstrates automatic type conversion using registered codecs.
func GetTask(_ context.Context, req GetTaskRequest) (*GetTaskResponse, error) {
	// All parameters are automatically parsed:
	// - req.Path.Task: string "1" -> Task{ID: 1, Title: "Setup Development Environment"}
	// - req.Path.CompletedAt: string "2023-12-25T10:30:00Z" -> time.Time
	// - req.Query.Priority: string "high" -> Priority(2) [iota value]

	// Demonstrate benefits of iota-based enum:
	// 1. Type-safe comparisons
	var statusMessage string
	switch {
	case req.Query.Priority >= PriorityHigh:
		statusMessage = "High priority task - needs immediate attention!"
	case req.Query.Priority == PriorityMedium:
		statusMessage = "Medium priority task - scheduled for this week"
	default:
		statusMessage = "Low priority task - backlog item"
	}

	// 2. Efficient ordering and comparison
	priorityLevel := int(req.Query.Priority) // Can use numeric value for sorting/ordering

	return &GetTaskResponse{
		Body: struct {
			Task        Task      `gork:"task"`
			Priority    Priority  `gork:"priority"`
			CompletedAt time.Time `gork:"completedAt"`
			Message     string    `gork:"message"`
		}{
			Task:        req.Path.Task,        // Full task entity available
			Priority:    req.Query.Priority,   // Validated enum value (iota internally, string externally)
			CompletedAt: req.Path.CompletedAt, // Parsed time value
			Message:     fmt.Sprintf("Task '%s' (level %d) with %s completed at %s", req.Path.Task.Title, priorityLevel, statusMessage, req.Path.CompletedAt.Format("2006-01-02 15:04")),
		},
	}, nil
}

// Helper function for creating pointers.
func ptr[T any](v T) *T {
	return &v
}

// Register codecs when package is imported. time.Time needs no registration.
func init() {
	if err := gorkson.RegisterCodec[Priority](PriorityCodec{}); err != nil {
		panic(fmt.Sprintf("failed to register Priority codec: %v", err))
	}
	if err := gorkson.RegisterCodec[Task](TaskCodec{}); err != nil {
		panic(fmt.Sprintf("failed to register Task codec: %v", err))
	}
}
