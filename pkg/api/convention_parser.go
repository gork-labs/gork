package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gork-labs/gork/pkg/gorkson"
)

// Standard section names as defined in the Convention Over Configuration spec.
const (
	SectionQuery   = "Query"
	SectionBody    = "Body"
	SectionPath    = "Path"
	SectionHeaders = "Headers"
	SectionCookies = "Cookies"
)

// AllowedSections defines the valid section names.
var AllowedSections = map[string]bool{
	SectionQuery:   true,
	SectionBody:    true,
	SectionPath:    true,
	SectionHeaders: true,
	SectionCookies: true,
}

// ConventionParser handles parsing requests using the Convention Over Configuration approach.
type ConventionParser struct {
	typeRegistry *TypeParserRegistry
	validator    *validator.Validate
}

// NewConventionParser creates a new convention parser.
func NewConventionParser() *ConventionParser {
	return &ConventionParser{
		typeRegistry: NewTypeParserRegistry(),
		validator:    validator.New(),
	}
}

// ParseRequest provides a public API for parsing HTTP requests using convention over configuration.
// This is the main entry point for webhook handlers and other use cases that need request parsing.
func ParseRequest(r *http.Request, reqPtr interface{}) error {
	parser := NewConventionParser()

	// Create a default parameter adapter that extracts from standard HTTP request
	adapter := NewDefaultParameterAdapter()

	reqValue := reflect.ValueOf(reqPtr)
	if reqValue.Kind() != reflect.Ptr {
		return fmt.Errorf("request must be a pointer")
	}

	return parser.ParseRequest(r.Context(), r, reqValue, adapter)
}

// DefaultParameterAdapter provides basic parameter extraction from http.Request
// without framework-specific functionality. Used by the public ParseRequest API.
type DefaultParameterAdapter struct{}

// NewDefaultParameterAdapter creates a new default parameter adapter.
func NewDefaultParameterAdapter() *DefaultParameterAdapter {
	return &DefaultParameterAdapter{}
}

// Path extracts path parameters - limited without framework router.
func (d *DefaultParameterAdapter) Path(_ *http.Request, _ string) (string, bool) {
	// Without a framework router, we cannot extract path parameters
	// This would need to be implemented by framework-specific adapters
	return "", false
}

// Query extracts query parameters from the URL.
func (d *DefaultParameterAdapter) Query(r *http.Request, key string) (string, bool) {
	value := r.URL.Query().Get(key)
	return value, value != ""
}

// Header extracts headers from the request.
func (d *DefaultParameterAdapter) Header(r *http.Request, key string) (string, bool) {
	value := r.Header.Get(key)
	return value, value != ""
}

// Cookie extracts cookies from the request.
func (d *DefaultParameterAdapter) Cookie(r *http.Request, key string) (string, bool) {
	cookie, err := r.Cookie(key)
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}

// RegisterTypeParser registers a type parser function.
func (p *ConventionParser) RegisterTypeParser(parserFunc interface{}) error {
	return p.typeRegistry.Register(parserFunc)
}

// ParseRequest parses an HTTP request into the given request struct using convention over configuration.
// Follows spec parsing order: Path, Query, Headers, Cookies, Body.
func (p *ConventionParser) ParseRequest(ctx context.Context, r *http.Request, reqPtr reflect.Value, adapter GenericParameterAdapter[*http.Request]) error {
	if reqPtr.Kind() != reflect.Ptr || reqPtr.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("request must be a pointer to struct")
	}

	reqStruct := reqPtr.Elem()
	reqType := reqStruct.Type()

	// Parse sections in the order specified by the spec
	sectionOrder := []string{SectionPath, SectionQuery, SectionHeaders, SectionCookies, SectionBody}

	for _, sectionName := range sectionOrder {
		field, fieldValue := p.findSection(reqType, reqStruct, sectionName)
		if field != nil {
			if err := p.parseSection(ctx, sectionName, fieldValue, r, adapter); err != nil {
				return fmt.Errorf("failed to parse %s section: %w", sectionName, err)
			}
		}
	}

	return nil
}

// findSection finds a section field in the struct.
func (p *ConventionParser) findSection(reqType reflect.Type, reqStruct reflect.Value, sectionName string) (*reflect.StructField, reflect.Value) {
	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		if field.Name == sectionName {
			return &field, reqStruct.Field(i)
		}
	}
	return nil, reflect.Value{}
}

// parseSection parses a specific section of the request.
func (p *ConventionParser) parseSection(ctx context.Context, sectionName string, sectionValue reflect.Value, r *http.Request, adapter GenericParameterAdapter[*http.Request]) error {
	// Special case for Body field - allow []byte for raw body parsing (webhook support)
	if sectionName == SectionBody {
		return p.parseBodySection(sectionValue, r)
	}

	// All other sections must be structs
	if sectionValue.Kind() != reflect.Struct {
		return fmt.Errorf("section %s must be a struct", sectionName)
	}

	sectionType := sectionValue.Type()

	switch sectionName {
	case SectionPath:
		return p.parsePathSection(ctx, sectionValue, sectionType, r, adapter)
	case SectionQuery:
		return p.parseQuerySection(ctx, sectionValue, sectionType, r, adapter)
	case SectionHeaders:
		return p.parseHeadersSection(ctx, sectionValue, sectionType, r, adapter)
	case SectionCookies:
		return p.parseCookiesSection(ctx, sectionValue, sectionType, r, adapter)
	}

	return nil
}

// parseBodySection parses the request body using gork JSON or raw bytes.
func (p *ConventionParser) parseBodySection(sectionValue reflect.Value, r *http.Request) error {
	// Check if this is a direct []byte field instead of a struct
	if sectionValue.Kind() == reflect.Slice && sectionValue.Type().Elem().Kind() == reflect.Uint8 {
		return p.parseRawBodyField(sectionValue, r)
	}

	// Only parse body for methods that typically carry one for struct sections
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
		return nil
	}

	if r.Body == nil {
		return nil
	}

	// Read the body first
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Use gork JSON unmarshaling if body is not empty
	if len(bodyBytes) > 0 {
		// Create a pointer to the section struct for JSON decoding
		sectionPtr := reflect.New(sectionValue.Type())
		if err := gorkson.Unmarshal(bodyBytes, sectionPtr.Interface()); err != nil {
			return fmt.Errorf("failed to decode JSON body: %w", err)
		}

		// Copy the decoded values back to the original struct
		sectionValue.Set(sectionPtr.Elem())
	}
	return nil
}

// parseRawBodyField handles direct []byte Body fields for webhook support.
func (p *ConventionParser) parseRawBodyField(sectionValue reflect.Value, r *http.Request) error {
	if r.Body == nil {
		// Set empty slice for nil body
		sectionValue.Set(reflect.ValueOf([]byte{}))
		return nil
	}

	// Read the raw body bytes
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read raw request body: %w", err)
	}

	// Set the raw bytes directly
	sectionValue.Set(reflect.ValueOf(bodyBytes))
	return nil
}

// parsePathSection parses path parameters.
func (p *ConventionParser) parsePathSection(ctx context.Context, sectionValue reflect.Value, sectionType reflect.Type, r *http.Request, adapter GenericParameterAdapter[*http.Request]) error {
	if adapter == nil {
		return nil
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		fieldValue := sectionValue.Field(i)

		gorkTag := field.Tag.Get("gork")
		if gorkTag == "" {
			continue
		}

		paramName := parseGorkTag(gorkTag).Name
		if val, ok := adapter.Path(r, paramName); ok {
			if err := p.setFieldValue(ctx, fieldValue, field, val); err != nil {
				return fmt.Errorf("failed to set path parameter %s: %w", paramName, err)
			}
		}
	}

	return nil
}

// parseQuerySection parses query parameters.
func (p *ConventionParser) parseQuerySection(ctx context.Context, sectionValue reflect.Value, sectionType reflect.Type, r *http.Request, adapter GenericParameterAdapter[*http.Request]) error {
	if adapter == nil {
		return nil
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		fieldValue := sectionValue.Field(i)

		gorkTag := field.Tag.Get("gork")
		if gorkTag == "" {
			continue
		}

		paramName := parseGorkTag(gorkTag).Name
		if val, ok := adapter.Query(r, paramName); ok {
			if err := p.setFieldValue(ctx, fieldValue, field, val); err != nil {
				return fmt.Errorf("failed to set query parameter %s: %w", paramName, err)
			}
		}
	}

	return nil
}

// parseHeadersSection parses HTTP headers.
func (p *ConventionParser) parseHeadersSection(ctx context.Context, sectionValue reflect.Value, sectionType reflect.Type, r *http.Request, adapter GenericParameterAdapter[*http.Request]) error {
	if adapter == nil {
		return nil
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		fieldValue := sectionValue.Field(i)

		gorkTag := field.Tag.Get("gork")
		if gorkTag == "" {
			continue
		}

		headerName := parseGorkTag(gorkTag).Name
		if val, ok := adapter.Header(r, headerName); ok {
			if err := p.setFieldValue(ctx, fieldValue, field, val); err != nil {
				return fmt.Errorf("failed to set header %s: %w", headerName, err)
			}
		}
	}

	return nil
}

// parseCookiesSection parses HTTP cookies.
func (p *ConventionParser) parseCookiesSection(ctx context.Context, sectionValue reflect.Value, sectionType reflect.Type, r *http.Request, adapter GenericParameterAdapter[*http.Request]) error {
	if adapter == nil {
		return nil
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		fieldValue := sectionValue.Field(i)

		gorkTag := field.Tag.Get("gork")
		if gorkTag == "" {
			continue
		}

		cookieName := parseGorkTag(gorkTag).Name
		if val, ok := adapter.Cookie(r, cookieName); ok {
			if err := p.setFieldValue(ctx, fieldValue, field, val); err != nil {
				return fmt.Errorf("failed to set cookie %s: %w", cookieName, err)
			}
		}
	}

	return nil
}

// setFieldValue sets a field value with type conversion and complex type parsing.
func (p *ConventionParser) setFieldValue(ctx context.Context, fieldValue reflect.Value, field reflect.StructField, value string) error {
	// First try complex type parsing
	if parser := p.typeRegistry.GetParser(field.Type); parser != nil {
		result, err := parser(ctx, value)
		if err != nil {
			return err
		}
		fieldValue.Set(reflect.ValueOf(result).Elem())
		return nil
	}

	// Fall back to basic type conversion
	return p.setBasicFieldValue(fieldValue, field, value)
}

// setBasicFieldValue handles basic type conversions.
func (p *ConventionParser) setBasicFieldValue(fieldValue reflect.Value, field reflect.StructField, value string) error {
	kind := field.Type.Kind()
	if p.isBasicKind(kind) {
		return p.setBasicFieldValueForKind(fieldValue, kind, value)
	}

	// Handle special cases
	if kind == reflect.Slice {
		return p.setSliceFieldValue(fieldValue, field, value)
	}

	return p.setSpecialFieldValue(fieldValue, field, value)
}

// isBasicKind checks if the kind is a basic type that can be converted directly.
func (p *ConventionParser) isBasicKind(kind reflect.Kind) bool {
	return kind == reflect.String ||
		kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 ||
		kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 ||
		kind == reflect.Bool || kind == reflect.Float32 || kind == reflect.Float64
}

// setBasicFieldValueForKind handles basic type conversions for specific kinds.
func (p *ConventionParser) setBasicFieldValueForKind(fieldValue reflect.Value, kind reflect.Kind, value string) error {
	if kind == reflect.String {
		fieldValue.SetString(value)
		return nil
	}
	if kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 {
		return p.setIntFieldValue(fieldValue, value)
	}
	if kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 {
		return p.setUintFieldValue(fieldValue, value)
	}
	if kind == reflect.Bool {
		return p.setBoolFieldValue(fieldValue, value)
	}
	if kind == reflect.Float32 || kind == reflect.Float64 {
		return p.setFloatFieldValue(fieldValue, value)
	}
	return fmt.Errorf("unsupported field type: %s", kind)
}

// setIntFieldValue handles integer field conversions.
func (p *ConventionParser) setIntFieldValue(fieldValue reflect.Value, value string) error {
	iv, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid integer value: %s", value)
	}
	fieldValue.SetInt(iv)
	return nil
}

// setUintFieldValue handles unsigned integer field conversions.
func (p *ConventionParser) setUintFieldValue(fieldValue reflect.Value, value string) error {
	uv, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid unsigned integer value: %s", value)
	}
	fieldValue.SetUint(uv)
	return nil
}

// setBoolFieldValue handles boolean field conversions.
func (p *ConventionParser) setBoolFieldValue(fieldValue reflect.Value, value string) error {
	bv, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("invalid boolean value: %s", value)
	}
	fieldValue.SetBool(bv)
	return nil
}

// setFloatFieldValue handles float field conversions.
func (p *ConventionParser) setFloatFieldValue(fieldValue reflect.Value, value string) error {
	fv, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid float value: %s", value)
	}
	fieldValue.SetFloat(fv)
	return nil
}

// setSpecialFieldValue handles special types like time.Time.
func (p *ConventionParser) setSpecialFieldValue(fieldValue reflect.Value, field reflect.StructField, value string) error {
	// Try to handle time.Time specially
	if field.Type == reflect.TypeOf(time.Time{}) {
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return fmt.Errorf("invalid time format: %s", value)
		}
		fieldValue.Set(reflect.ValueOf(t))
		return nil
	}
	return fmt.Errorf("unsupported field type: %s", field.Type.Kind())
}

// setSliceFieldValue handles slice field conversions - simplified for string slices only.
func (p *ConventionParser) setSliceFieldValue(fieldValue reflect.Value, field reflect.StructField, value string) error {
	if field.Type.Elem().Kind() != reflect.String {
		return fmt.Errorf("only string slices are supported in query/path/header parameters")
	}

	if value == "" {
		return nil // Empty value, leave slice as zero value
	}

	// Simple comma-separated parsing
	parts := strings.Split(value, ",")
	sliceVal := reflect.MakeSlice(field.Type, len(parts), len(parts))
	for idx, part := range parts {
		sliceVal.Index(idx).SetString(strings.TrimSpace(part))
	}
	fieldValue.Set(sliceVal)
	return nil
}

// GorkTagInfo represents parsed gork tag information.
type GorkTagInfo struct {
	Name          string
	Discriminator string
}

// parseGorkTag parses a gork tag: "field_name[,discriminator=value,...]".
func parseGorkTag(tag string) GorkTagInfo {
	var info GorkTagInfo
	if tag == "" {
		return info
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		info.Name = strings.TrimSpace(parts[0])
	}

	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			if key == "discriminator" {
				info.Discriminator = val
			}
		}
	}

	return info
}
