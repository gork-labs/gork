package gorkson

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// Global codec registry for the gorkson package.
var globalCodecRegistry = NewCodecRegistry()

func init() {
	_ = RegisterCodec[time.Time](TimeCodec{})
}

// RegisterCodec registers a codec for type T in the global registry.
// A later registration for the same type replaces the earlier codec.
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

// parseWithCodec validates value against the codec schema of t and parses it with parser.
func parseWithCodec(ctx context.Context, t reflect.Type, parser TypeParser, value string) (reflect.Value, error) {
	schema, _ := globalCodecRegistry.Schema(t)
	if err := validateValueAgainstSchema(value, schema, t.String()); err != nil {
		return reflect.Value{}, err
	}

	result, err := parser(ctx, value)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(result).Elem(), nil
}

// formatWithCodec formats val with formatter, which expects a pointer to the value.
func formatWithCodec(ctx context.Context, val reflect.Value, formatter TypeFormatter) (string, error) {
	ptr := reflect.New(val.Type())
	ptr.Elem().Set(val)
	return formatter(ctx, ptr.Interface())
}

// isJSONTextSchema reports whether the codec text for t is a JSON value itself
// (a number, a boolean, an object or an array) and not the content of a JSON string.
func isJSONTextSchema(t reflect.Type) bool {
	schema, _ := globalCodecRegistry.Schema(t)
	return schema.Type != "" && schema.Type != OpenAPITypeString
}

// formatJSONWithCodec formats val with its codec for a JSON document.
func formatJSONWithCodec(val reflect.Value, formatter TypeFormatter) (any, error) {
	text, err := formatWithCodec(context.Background(), val, formatter)
	if err != nil {
		return nil, err
	}
	if isJSONTextSchema(val.Type()) {
		return json.RawMessage(text), nil
	}
	return text, nil
}

// setJSONValueWithCodec parses a decoded JSON value with the codec of the field type.
// A JSON string gives its content to the codec. Other JSON values give their JSON text.
func setJSONValueWithCodec(field reflect.Value, parser TypeParser, value any) error {
	text, isString := value.(string)
	if !isString {
		// A value that encoding/json decoded always encodes again.
		data, _ := json.Marshal(value)
		text = string(data)
	}

	parsed, err := parseWithCodec(context.Background(), field.Type(), parser, text)
	if err != nil {
		return err
	}
	field.Set(parsed)
	return nil
}
