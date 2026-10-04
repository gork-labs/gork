package gorkson

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

// useTestCodecRegistry gives the test a global registry with only the default codecs.
func useTestCodecRegistry(t *testing.T) {
	t.Helper()
	original := globalCodecRegistry
	globalCodecRegistry = NewCodecRegistry()
	if err := RegisterCodec[time.Time](TimeCodec{}); err != nil {
		t.Fatalf("RegisterCodec(TimeCodec) error = %v", err)
	}
	t.Cleanup(func() { globalCodecRegistry = original })
}

// Level is a test type with an integer codec.
type Level struct {
	Value int
}

type levelCodec struct{}

func (levelCodec) Parse(_ context.Context, value string) (*Level, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return nil, NewParseError("Level", value, err)
	}
	return &Level{Value: n}, nil
}

func (levelCodec) Format(_ context.Context, value *Level) (string, error) {
	if value.Value < 0 {
		return "", NewFormatError("Level", "negative level")
	}
	return strconv.Itoa(value.Value), nil
}

func (levelCodec) Schema() OpenAPISchema {
	return OpenAPISchema{Type: OpenAPITypeInteger, Minimum: ptr(0.0)}
}

// Code is a test type with a string codec that has schema constraints.
type Code struct {
	Value string
}

type codeCodec struct{}

func (codeCodec) Parse(_ context.Context, value string) (*Code, error) {
	if value == "XX" {
		return nil, errors.New("reserved code")
	}
	return &Code{Value: value}, nil
}

func (codeCodec) Format(_ context.Context, value *Code) (string, error) {
	return value.Value, nil
}

func (codeCodec) Schema() OpenAPISchema {
	return OpenAPISchema{Type: OpenAPITypeString, Pattern: `^[A-Z]{2}$`}
}
