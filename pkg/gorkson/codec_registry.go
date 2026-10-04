package gorkson

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// codecRegistry manages type codecs for different types.
type codecRegistry struct {
	mu      sync.RWMutex
	entries map[reflect.Type]TypeCodecEntry
}

// NewCodecRegistry creates a new codec registry.
func NewCodecRegistry() CodecRegistry {
	return &codecRegistry{
		entries: make(map[reflect.Type]TypeCodecEntry),
	}
}

// RegisterCodec registers a codec using type-erased interface.
// It replaces an earlier codec for the same type.
func (r *codecRegistry) RegisterCodec(targetType reflect.Type, parser TypeParser, formatter TypeFormatter, schema OpenAPISchema) error {
	if targetType == nil {
		return fmt.Errorf("target type cannot be nil")
	}
	if parser == nil {
		return fmt.Errorf("parser cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[targetType] = TypeCodecEntry{
		Parser:    parser,
		Formatter: formatter,
		Schema:    schema,
	}
	return nil
}

// Parser returns the parser for the given type.
func (r *codecRegistry) Parser(targetType reflect.Type) (func(ctx context.Context, value string) (interface{}, error), bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[targetType]
	if !exists {
		return nil, false
	}
	return entry.Parser, true
}

// Formatter returns the formatter for the given type.
func (r *codecRegistry) Formatter(targetType reflect.Type) (func(ctx context.Context, value interface{}) (string, error), bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[targetType]
	if !exists || entry.Formatter == nil {
		return nil, false
	}
	return entry.Formatter, true
}

// Schema returns the schema for the given type.
func (r *codecRegistry) Schema(targetType reflect.Type) (OpenAPISchema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[targetType]
	if !exists {
		return OpenAPISchema{}, false
	}
	return entry.Schema, true
}

// HasParser returns true if a parser is registered for the given type.
func (r *codecRegistry) HasParser(targetType reflect.Type) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.entries[targetType]
	return exists
}
