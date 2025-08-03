package api

import (
	"context"
	"fmt"
	"reflect"
)

// TypeParserFunc represents a type parser function with signature:
// func(ctx context.Context, value string) (*T, error).
type TypeParserFunc func(context.Context, string) (interface{}, error)

// TypeParserRegistry manages type parsers for complex types.
type TypeParserRegistry struct {
	parsers map[reflect.Type]TypeParserFunc
}

// NewTypeParserRegistry creates a new type parser registry.
func NewTypeParserRegistry() *TypeParserRegistry {
	return &TypeParserRegistry{
		parsers: make(map[reflect.Type]TypeParserFunc),
	}
}

// Register registers a type parser function.
// The function must have the signature: func(ctx context.Context, value string) (*T, error).
func (r *TypeParserRegistry) Register(parserFunc interface{}) error {
	funcValue := reflect.ValueOf(parserFunc)
	funcType := funcValue.Type()

	// Validate function signature
	if err := r.validateParserSignature(funcType); err != nil {
		return err
	}

	// Extract the target type from the return value
	targetType := funcType.Out(0).Elem() // *T -> T

	// Create a wrapper function that calls the original with proper type conversion
	wrapper := func(ctx context.Context, value string) (interface{}, error) {
		results := funcValue.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(value),
		})

		result := results[0].Interface()
		errInterface := results[1].Interface()

		if errInterface != nil {
			if err, ok := errInterface.(error); ok {
				return nil, err
			}
		}

		return result, nil
	}

	r.parsers[targetType] = wrapper
	return nil
}

// validateParserSignature validates that the parser function has the correct signature.
func (r *TypeParserRegistry) validateParserSignature(funcType reflect.Type) error {
	if funcType.Kind() != reflect.Func {
		return fmt.Errorf("parser must be a function")
	}

	if funcType.NumIn() != 2 {
		return fmt.Errorf("parser must accept exactly 2 parameters (context.Context, string)")
	}

	// Check first parameter is context.Context
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !funcType.In(0).Implements(contextType) {
		return fmt.Errorf("first parameter must be context.Context")
	}

	// Check second parameter is string
	if funcType.In(1).Kind() != reflect.String {
		return fmt.Errorf("second parameter must be string")
	}

	if funcType.NumOut() != 2 {
		return fmt.Errorf("parser must return exactly 2 values (*T, error)")
	}

	// Check first return value is a pointer
	if funcType.Out(0).Kind() != reflect.Ptr {
		return fmt.Errorf("first return value must be a pointer (*T)")
	}

	// Check second return value implements error
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !funcType.Out(1).Implements(errorType) {
		return fmt.Errorf("second return value must be error")
	}

	return nil
}

// GetParser returns the parser for the given type, if registered.
func (r *TypeParserRegistry) GetParser(targetType reflect.Type) TypeParserFunc {
	return r.parsers[targetType]
}

// HasParser returns true if a parser is registered for the given type.
func (r *TypeParserRegistry) HasParser(targetType reflect.Type) bool {
	_, exists := r.parsers[targetType]
	return exists
}

// ListRegisteredTypes returns a list of all registered types.
func (r *TypeParserRegistry) ListRegisteredTypes() []reflect.Type {
	types := make([]reflect.Type, 0, len(r.parsers))
	for t := range r.parsers {
		types = append(types, t)
	}
	return types
}
