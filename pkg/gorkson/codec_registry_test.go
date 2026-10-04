package gorkson

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestNewCodecRegistry(t *testing.T) {
	registry := NewCodecRegistry()
	if registry == nil {
		t.Fatal("NewCodecRegistry() returned nil")
	}
}

func TestCodecRegistry_RegisterCodec(t *testing.T) {
	dummyParser := func(ctx context.Context, value string) (interface{}, error) {
		return &value, nil
	}
	dummyFormatter := func(ctx context.Context, value interface{}) (string, error) {
		return "formatted", nil
	}
	schema := OpenAPISchema{Type: "string"}

	tests := []struct {
		name        string
		targetType  reflect.Type
		parser      TypeParser
		formatter   TypeFormatter
		schema      OpenAPISchema
		wantErr     bool
		expectedErr string
	}{
		{
			name:       "valid registration",
			targetType: reflect.TypeOf(""),
			parser:     dummyParser,
			formatter:  dummyFormatter,
			schema:     schema,
			wantErr:    false,
		},
		{
			name:        "nil target type",
			targetType:  nil,
			parser:      dummyParser,
			formatter:   dummyFormatter,
			schema:      schema,
			wantErr:     true,
			expectedErr: "target type cannot be nil",
		},
		{
			name:        "nil parser",
			targetType:  reflect.TypeOf(0), // Use int type to avoid conflict
			parser:      nil,
			formatter:   dummyFormatter,
			schema:      schema,
			wantErr:     true,
			expectedErr: "parser cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewCodecRegistry() // Create fresh registry for each test

			err := registry.RegisterCodec(tt.targetType, tt.parser, tt.formatter, tt.schema)

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterCodec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.expectedErr != "" && err.Error() != tt.expectedErr {
				t.Errorf("RegisterCodec() error = %v, expectedErr %v", err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestCodecRegistry_DuplicateRegistration(t *testing.T) {
	registry := NewCodecRegistry()

	dummyParser := func(ctx context.Context, value string) (interface{}, error) {
		return &value, nil
	}
	dummyFormatter := func(ctx context.Context, value interface{}) (string, error) {
		return "formatted", nil
	}
	schema := OpenAPISchema{Type: "string"}
	targetType := reflect.TypeOf("")

	// Register first time (should succeed)
	err := registry.RegisterCodec(targetType, dummyParser, dummyFormatter, schema)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Register same type again (should fail)
	err = registry.RegisterCodec(targetType, dummyParser, dummyFormatter, schema)
	if err == nil {
		t.Fatal("Expected duplicate registration to fail")
	}

	expectedMsg := "already registered"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestCodecRegistry_Parser(t *testing.T) {
	registry := NewCodecRegistry()

	dummyParser := func(ctx context.Context, value string) (interface{}, error) {
		return &value, nil
	}
	dummyFormatter := func(ctx context.Context, value interface{}) (string, error) {
		return "formatted", nil
	}
	schema := OpenAPISchema{Type: "string"}

	stringType := reflect.TypeOf("")
	intType := reflect.TypeOf(0)

	// Register parser for string type
	err := registry.RegisterCodec(stringType, dummyParser, dummyFormatter, schema)
	if err != nil {
		t.Fatalf("Failed to register codec: %v", err)
	}

	tests := []struct {
		name       string
		targetType reflect.Type
		wantExists bool
	}{
		{
			name:       "existing parser",
			targetType: stringType,
			wantExists: true,
		},
		{
			name:       "non-existing parser",
			targetType: intType,
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, exists := registry.Parser(tt.targetType)

			if exists != tt.wantExists {
				t.Errorf("Parser() exists = %v, wantExists %v", exists, tt.wantExists)
				return
			}

			if tt.wantExists && parser == nil {
				t.Error("Parser() returned nil parser when expected to exist")
			}

			if !tt.wantExists && parser != nil {
				t.Error("Parser() returned non-nil parser when expected not to exist")
			}
		})
	}
}

func TestCodecRegistry_Formatter(t *testing.T) {
	registry := NewCodecRegistry()

	dummyParser := func(ctx context.Context, value string) (interface{}, error) {
		return &value, nil
	}
	dummyFormatter := func(ctx context.Context, value interface{}) (string, error) {
		return "formatted", nil
	}
	schema := OpenAPISchema{Type: "string"}

	stringType := reflect.TypeOf("")
	intType := reflect.TypeOf(0)

	// Register codec with formatter for string type
	err := registry.RegisterCodec(stringType, dummyParser, dummyFormatter, schema)
	if err != nil {
		t.Fatalf("Failed to register codec: %v", err)
	}

	// Register codec without formatter for int type
	err = registry.RegisterCodec(intType, dummyParser, nil, schema)
	if err != nil {
		t.Fatalf("Failed to register codec: %v", err)
	}

	tests := []struct {
		name       string
		targetType reflect.Type
		wantExists bool
	}{
		{
			name:       "existing formatter",
			targetType: stringType,
			wantExists: true,
		},
		{
			name:       "no formatter",
			targetType: intType,
			wantExists: false,
		},
		{
			name:       "non-existing codec",
			targetType: reflect.TypeOf(float64(0)),
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, exists := registry.Formatter(tt.targetType)

			if exists != tt.wantExists {
				t.Errorf("Formatter() exists = %v, wantExists %v", exists, tt.wantExists)
				return
			}

			if tt.wantExists && formatter == nil {
				t.Error("Formatter() returned nil formatter when expected to exist")
			}

			if !tt.wantExists && formatter != nil {
				t.Error("Formatter() returned non-nil formatter when expected not to exist")
			}
		})
	}
}

func TestCodecRegistry_Schema(t *testing.T) {
	registry := NewCodecRegistry()

	dummyParser := func(ctx context.Context, value string) (interface{}, error) {
		return &value, nil
	}
	schema := OpenAPISchema{Type: "string", Format: "custom"}

	stringType := reflect.TypeOf("")
	intType := reflect.TypeOf(0)

	// Register codec for string type
	err := registry.RegisterCodec(stringType, dummyParser, nil, schema)
	if err != nil {
		t.Fatalf("Failed to register codec: %v", err)
	}

	tests := []struct {
		name         string
		targetType   reflect.Type
		wantExists   bool
		expectedType string
	}{
		{
			name:         "existing schema",
			targetType:   stringType,
			wantExists:   true,
			expectedType: "string",
		},
		{
			name:       "non-existing schema",
			targetType: intType,
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultSchema, exists := registry.Schema(tt.targetType)

			if exists != tt.wantExists {
				t.Errorf("Schema() exists = %v, wantExists %v", exists, tt.wantExists)
				return
			}

			if tt.wantExists && resultSchema.Type != tt.expectedType {
				t.Errorf("Schema() type = %v, expectedType %v", resultSchema.Type, tt.expectedType)
			}
		})
	}
}

func TestCodecRegistry_HasParser(t *testing.T) {
	registry := NewCodecRegistry()

	dummyParser := func(ctx context.Context, value string) (interface{}, error) {
		return &value, nil
	}
	schema := OpenAPISchema{Type: "string"}

	stringType := reflect.TypeOf("")
	intType := reflect.TypeOf(0)

	// Register parser for string type
	err := registry.RegisterCodec(stringType, dummyParser, nil, schema)
	if err != nil {
		t.Fatalf("Failed to register codec: %v", err)
	}

	tests := []struct {
		name       string
		targetType reflect.Type
		want       bool
	}{
		{
			name:       "has parser",
			targetType: stringType,
			want:       true,
		},
		{
			name:       "no parser",
			targetType: intType,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registry.HasParser(tt.targetType)
			if got != tt.want {
				t.Errorf("HasParser() = %v, want %v", got, tt.want)
			}
		})
	}
}
