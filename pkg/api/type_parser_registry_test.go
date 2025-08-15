package api

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// Test types for type parser
type TestUser struct {
	ID   string
	Name string
}

type TestCompany struct {
	ID   string
	Name string
}

func TestTypeParserRegistry_Register(t *testing.T) {
	registry := NewTypeParserRegistry()

	tests := []struct {
		name        string
		parserFunc  interface{}
		wantErr     bool
		expectedErr string
	}{
		{
			name: "valid parser function",
			parserFunc: func(ctx context.Context, value string) (*TestUser, error) {
				return &TestUser{ID: value, Name: "Test User"}, nil
			},
			wantErr: false,
		},
		{
			name: "valid time parser",
			parserFunc: func(ctx context.Context, value string) (*time.Time, error) {
				// Use deterministic time parsing for reliable tests
				t, err := time.Parse(time.RFC3339, value)
				return &t, err
			},
			wantErr: false,
		},
		{
			name:        "invalid - not a function",
			parserFunc:  "not a function",
			wantErr:     true,
			expectedErr: "parser must be a function",
		},
		{
			name: "invalid - wrong number of parameters",
			parserFunc: func(value string) (*TestUser, error) {
				return &TestUser{}, nil
			},
			wantErr:     true,
			expectedErr: "parser must accept exactly 2 parameters (context.Context, string)",
		},
		{
			name: "invalid - first parameter not context",
			parserFunc: func(value string, ctx context.Context) (*TestUser, error) {
				return &TestUser{}, nil
			},
			wantErr:     true,
			expectedErr: "first parameter must be context.Context",
		},
		{
			name: "invalid - second parameter not string",
			parserFunc: func(ctx context.Context, value int) (*TestUser, error) {
				return &TestUser{}, nil
			},
			wantErr:     true,
			expectedErr: "second parameter must be string",
		},
		{
			name: "invalid - wrong number of returns",
			parserFunc: func(ctx context.Context, value string) *TestUser {
				return &TestUser{}
			},
			wantErr:     true,
			expectedErr: "parser must return exactly 2 values (*T, error)",
		},
		{
			name: "invalid - first return not pointer",
			parserFunc: func(ctx context.Context, value string) (TestUser, error) {
				return TestUser{}, nil
			},
			wantErr:     true,
			expectedErr: "first return value must be a pointer (*T)",
		},
		{
			name: "invalid - second return not error",
			parserFunc: func(ctx context.Context, value string) (*TestUser, string) {
				return &TestUser{}, ""
			},
			wantErr:     true,
			expectedErr: "second return value must be error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.parserFunc)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if err.Error() != tt.expectedErr {
					t.Errorf("Register() error = %v, expectedErr %v", err.Error(), tt.expectedErr)
				}
			}
		})
	}
}

func TestTypeParserRegistry_GetParser(t *testing.T) {
	registry := NewTypeParserRegistry()

	// Register a parser for TestUser
	userParser := func(ctx context.Context, value string) (*TestUser, error) {
		if value == "error" {
			return nil, errors.New("test error")
		}
		return &TestUser{ID: value, Name: "Test User " + value}, nil
	}

	err := registry.Register(userParser)
	if err != nil {
		t.Fatalf("Failed to register parser: %v", err)
	}

	tests := []struct {
		name       string
		targetType reflect.Type
		hasParser  bool
	}{
		{
			name:       "registered type",
			targetType: reflect.TypeOf(TestUser{}),
			hasParser:  true,
		},
		{
			name:       "unregistered type",
			targetType: reflect.TypeOf(TestCompany{}),
			hasParser:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := registry.GetParser(tt.targetType)
			hasParser := parser != nil

			if hasParser != tt.hasParser {
				t.Errorf("GetParser() hasParser = %v, want %v", hasParser, tt.hasParser)
			}

			// Test the parser if it exists
			if hasParser {
				result, err := parser(context.Background(), "test-id")
				if err != nil {
					t.Errorf("Parser returned error: %v", err)
				}

				if user, ok := result.(*TestUser); ok {
					if user.ID != "test-id" {
						t.Errorf("Parser result ID = %v, want %v", user.ID, "test-id")
					}
				} else {
					t.Errorf("Parser result type = %T, want *TestUser", result)
				}

				// Test error case
				_, err = parser(context.Background(), "error")
				if err == nil {
					t.Errorf("Parser should return error for 'error' input")
				}
			}
		})
	}
}

func TestTypeParserRegistry_HasParser(t *testing.T) {
	registry := NewTypeParserRegistry()

	// Register a parser
	err := registry.Register(func(ctx context.Context, value string) (*TestUser, error) {
		return &TestUser{}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register parser: %v", err)
	}

	tests := []struct {
		name       string
		targetType reflect.Type
		want       bool
	}{
		{
			name:       "registered type",
			targetType: reflect.TypeOf(TestUser{}),
			want:       true,
		},
		{
			name:       "unregistered type",
			targetType: reflect.TypeOf(TestCompany{}),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registry.HasParser(tt.targetType); got != tt.want {
				t.Errorf("HasParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeParserRegistry_ListRegisteredTypes(t *testing.T) {
	registry := NewTypeParserRegistry()

	// Register multiple parsers
	err := registry.Register(func(ctx context.Context, value string) (*TestUser, error) {
		return &TestUser{}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register TestUser parser: %v", err)
	}

	err = registry.Register(func(ctx context.Context, value string) (*time.Time, error) {
		// Use a fixed time for deterministic testing
		fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		return &fixedTime, nil
	})
	if err != nil {
		t.Fatalf("Failed to register time.Time parser: %v", err)
	}

	types := registry.ListRegisteredTypes()

	if len(types) != 2 {
		t.Errorf("ListRegisteredTypes() returned %d types, want 2", len(types))
	}

	// Check that both types are present
	typeNames := make(map[string]bool)
	for _, t := range types {
		typeNames[t.Name()] = true
	}

	if !typeNames["TestUser"] {
		t.Errorf("TestUser type not found in registered types")
	}

	if !typeNames["Time"] {
		t.Errorf("Time type not found in registered types")
	}
}
