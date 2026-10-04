package gorkson

import (
	"context"
	"reflect"
)

// TypeCodec handles bidirectional conversion and provides OpenAPI schema.
//
// gorkson uses the codec for parameters, JSON request bodies and JSON responses.
// In JSON, the text of a codec with a "string" schema type is a JSON string.
// For other schema types the text is the JSON value itself, for example 123 or true.
type TypeCodec[T any] interface {
	// Parse converts the text form of a value to the Go type
	Parse(ctx context.Context, value string) (*T, error)

	// Format converts the Go type to its text form
	Format(ctx context.Context, value *T) (string, error)

	// Schema provides OpenAPI schema information for this type.
	// Parse receives only values that pass the pattern, length, range and enum constraints of the schema.
	Schema() OpenAPISchema
}

// OpenAPI type constants.
const (
	OpenAPITypeString  = "string"
	OpenAPITypeInteger = "integer"
	OpenAPITypeNumber  = "number"
	OpenAPITypeBoolean = "boolean"
	OpenAPITypeArray   = "array"
	OpenAPITypeObject  = "object"
)

// OpenAPI format constants.
const (
	OpenAPIFormatUUID     = "uuid"
	OpenAPIFormatDateTime = "date-time"
	OpenAPIFormatDuration = "duration"
	OpenAPIFormatEmail    = "email"
	OpenAPIFormatURI      = "uri"
	OpenAPIFormatByte     = "byte"
	OpenAPIFormatBinary   = "binary"
)

// OpenAPISchema represents OpenAPI schema information for a type codec.
type OpenAPISchema struct {
	Type        string                    `json:"type"` // "string", "integer", "number", "boolean", "array", "object"
	Format      string                    `json:"format,omitempty"`
	Pattern     string                    `json:"pattern,omitempty"`
	Example     interface{}               `json:"example,omitempty"`
	Description string                    `json:"description,omitempty"`
	MinLength   *int                      `json:"minLength,omitempty"`
	MaxLength   *int                      `json:"maxLength,omitempty"`
	Minimum     *float64                  `json:"minimum,omitempty"` // For numeric types
	Maximum     *float64                  `json:"maximum,omitempty"` // For numeric types
	Enum        []interface{}             `json:"enum,omitempty"`
	Properties  map[string]*OpenAPISchema `json:"properties,omitempty"` // For object types
	Items       *OpenAPISchema            `json:"items,omitempty"`      // For array types
}

// TypeCodecEntry contains codec information with type-erased methods.
type TypeCodecEntry struct {
	Parser    func(ctx context.Context, value string) (interface{}, error)
	Formatter func(ctx context.Context, value interface{}) (string, error)
	Schema    OpenAPISchema
}

// CodecRegistry manages type codecs for different types.
type CodecRegistry interface {
	// RegisterCodec registers a codec using type-erased interface
	RegisterCodec(targetType reflect.Type, parser TypeParser, formatter TypeFormatter, schema OpenAPISchema) error

	// Parser returns the parser for the given type
	Parser(targetType reflect.Type) (func(ctx context.Context, value string) (interface{}, error), bool)

	// Formatter returns the formatter for the given type
	Formatter(targetType reflect.Type) (func(ctx context.Context, value interface{}) (string, error), bool)

	// Schema returns the schema for the given type
	Schema(targetType reflect.Type) (OpenAPISchema, bool)

	// HasParser returns true if a parser is registered for the given type
	HasParser(targetType reflect.Type) bool
}

// TypeParser represents a parser function for a specific type.
type TypeParser func(ctx context.Context, value string) (interface{}, error)

// TypeFormatter represents a formatter function for a specific type.
type TypeFormatter func(ctx context.Context, value interface{}) (string, error)
