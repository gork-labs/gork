package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

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
	validator *validator.Validate
}

// NewConventionParser creates a new convention parser.
func NewConventionParser() *ConventionParser {
	return &ConventionParser{
		validator: validator.New(),
	}
}

// ParseRequest provides a public API for parsing HTTP requests using convention over configuration.
// This is the main entry point for webhook handlers and other use cases that need request parsing.
func ParseRequest(r *http.Request, reqPtr interface{}) error {
	parser := NewConventionParser()

	// Create a default parameter adapter that extracts from standard HTTP request
	adapter := NewDefaultParameterAdapter()

	reqValue := reflect.ValueOf(reqPtr)
	if reqValue.Kind() != reflect.Pointer {
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
func (d *DefaultParameterAdapter) Query(r *http.Request, key string) ([]string, bool) {
	value := r.URL.Query()[key]
	return value, len(value) > 0
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

// ParseRequest parses an HTTP request into the given request struct using convention over configuration.
// Follows spec parsing order: Path, Query, Headers, Cookies, Body.
func (p *ConventionParser) ParseRequest(ctx context.Context, r *http.Request, reqPtr reflect.Value, adapter GenericParameterAdapter[*http.Request]) error {
	if reqPtr.Kind() != reflect.Pointer || reqPtr.Elem().Kind() != reflect.Struct {
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

	if isMultipartBody(sectionValue.Type()) {
		return p.parseMultipartBody(r.Context(), sectionValue, r)
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

// parseMultipartBody reads the parts of a multipart/form-data body one at a time.
// A part for a File or []File field is a file part. Any other part is a text field.
// Each part for a []string field adds one item.
// A part with a name that no field has is skipped.
func (p *ConventionParser) parseMultipartBody(ctx context.Context, sectionValue reflect.Value, r *http.Request) error {
	reader, err := r.MultipartReader()
	if err != nil {
		return fmt.Errorf("invalid multipart body: %w", err)
	}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read multipart body: %w", err)
		}

		fieldValue, ok := p.findFormField(sectionValue, part.FormName())
		if !ok {
			continue
		}

		data, err := io.ReadAll(part)
		if err != nil {
			return fmt.Errorf("failed to read form field %s: %w", part.FormName(), err)
		}

		switch fieldValue.Type() {
		case fileType:
			fieldValue.Set(reflect.ValueOf(newFile(part, data)))
		case fileSliceType:
			fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(newFile(part, data))))
		case stringSliceType:
			fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(string(data))))
		default:
			if err := gorkson.SetFieldValueFromString(ctx, fieldValue, string(data)); err != nil {
				return fmt.Errorf("failed to set form field %s: %w", part.FormName(), err)
			}
		}
	}
}

// findFormField finds the field of a Body struct with the given gork name.
func (p *ConventionParser) findFormField(sectionValue reflect.Value, name string) (reflect.Value, bool) {
	sectionType := sectionValue.Type()
	for i := 0; i < sectionType.NumField(); i++ {
		if tagName := parseGorkTag(sectionType.Field(i).Tag.Get("gork")).Name; tagName != "" && tagName == name {
			return sectionValue.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func newFile(part *multipart.Part, data []byte) File {
	return File{
		Name:        part.FileName(),
		ContentType: part.Header.Get("Content-Type"),
		Data:        data,
	}
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
			if err := gorkson.SetFieldValueFromString(ctx, fieldValue, val); err != nil {
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
		values, ok := adapter.Query(r, paramName)
		if !ok {
			continue
		}

		val := values[0]
		if fieldValue.Kind() == reflect.Slice {
			val = strings.Join(values, ",")
		}
		if val == "" {
			continue
		}

		if err := gorkson.SetFieldValueFromString(ctx, fieldValue, val); err != nil {
			return fmt.Errorf("failed to set query parameter %s: %w", paramName, err)
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
			if err := gorkson.SetFieldValueFromString(ctx, fieldValue, val); err != nil {
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
			if err := gorkson.SetFieldValueFromString(ctx, fieldValue, val); err != nil {
				return fmt.Errorf("failed to set cookie %s: %w", cookieName, err)
			}
		}
	}

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
