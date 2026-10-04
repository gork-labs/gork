package gorkson

import (
	"context"
	"fmt"
	"reflect"
)

// Global codec registry for the gorkson package.
var globalCodecRegistry = NewCodecRegistry()

// RegisterCodec provides a type-safe way to register a codec globally.
func RegisterCodec[T any](codec TypeCodec[T]) error {
	var zero T
	targetType := reflect.TypeOf(zero)

	// Create type-erased wrapper functions
	parser := func(ctx context.Context, value string) (interface{}, error) {
		result, err := codec.Parse(ctx, value)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	formatter := func(ctx context.Context, value interface{}) (string, error) {
		// Type assertion to *T
		typedValue, ok := value.(*T)
		if !ok {
			return "", fmt.Errorf("expected *%T, got %T", zero, value)
		}
		return codec.Format(ctx, typedValue)
	}

	return globalCodecRegistry.RegisterCodec(targetType, parser, formatter, codec.Schema())
}

// GetCodecRegistry returns the global codec registry.
func GetCodecRegistry() CodecRegistry {
	return globalCodecRegistry
}

// SetFieldValueFromString converts a string value to the appropriate Go type and sets it on the field.
// This is the main entry point for HTTP parameter parsing that delegates to codec system.
func SetFieldValueFromString(ctx context.Context, fieldValue reflect.Value, value string) error {
	fieldType := fieldValue.Type()

	// First try codec registry
	if parser, exists := globalCodecRegistry.Parser(fieldType); exists {
		result, err := parser(ctx, value)
		if err != nil {
			return err
		}
		fieldValue.Set(reflect.ValueOf(result).Elem())
		return nil
	}

	// Fall back to basic string conversion
	// TODO: This will be implemented when we enhance the main gorkson.Marshaler
	return fmt.Errorf("no codec registered for type %s", fieldType)
}
