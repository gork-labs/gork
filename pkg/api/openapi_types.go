package api

import "encoding/json"

// NOTE: These definitions intentionally keep only the fields we actively
// populate for now. Additional fields can be added without breaking existing
// users as we expand the generator.

// OpenAPISpec represents the root of an OpenAPI 3.1 document.
type OpenAPISpec struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`

	// routeFilter allows callers to skip specific RouteInfo entries during
	// spec generation. It is internal-only and therefore excluded from JSON
	// and YAML output.
	routeFilter func(*RouteInfo) bool `json:"-"`
}

// MarshalJSON implements a custom marshaler for OpenAPISpec to ensure that
// it is always marshaled using the standard json package, even when a custom
// marshaler (like gork's) is active.
func (s *OpenAPISpec) MarshalJSON() ([]byte, error) {
	type Alias OpenAPISpec
	return json.Marshal((*Alias)(s))
}

// Info represents the OpenAPI info section containing metadata about the API.
type Info struct {
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// Components represents the OpenAPI components section containing reusable objects.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"`
}

// PathItem represents a path item object containing HTTP operations for a path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation represents an OpenAPI operation object describing a single API operation.
type Operation struct {
	OperationID string                `json:"operationId,omitempty"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]*Response  `json:"responses,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty"`
}

// Parameter represents an OpenAPI parameter object describing a single operation parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // "query", "header", "path", "cookie"
	Required    bool    `json:"required"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody represents an OpenAPI request body object.
type RequestBody struct {
	Required bool                 `json:"required,omitempty"`
	Content  map[string]MediaType `json:"content,omitempty"`
}

// MediaType represents an OpenAPI media type object containing schema information.
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Response represents an OpenAPI response object describing a single response from an API operation.
type Response struct {
	Ref         string               `json:"$ref,omitempty"`
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Headers     map[string]*Header   `json:"headers,omitempty"`
}

// Header represents an OpenAPI header object.
type Header struct {
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Schema represents an OpenAPI schema object defining the structure of request/response data.
type Schema struct {
	// Title provides a human-readable name for the schema. Some documentation UIs
	// (e.g. ReDoc, Swagger UI) display this value in type signatures. We set it
	// automatically from the Go type name where available so that arrays like
	// []UserResponse are shown as array[UserResponse] instead of array[object].
	Title         string             `json:"title,omitempty"`
	Ref           string             `json:"$ref,omitempty"`
	Type          string             `json:"-"`
	Types         []string           `json:"-"`
	Properties    map[string]*Schema `json:"properties,omitempty"`
	Required      []string           `json:"required,omitempty"`
	OneOf         []*Schema          `json:"oneOf,omitempty"`
	AnyOf         []*Schema          `json:"anyOf,omitempty"`
	Discriminator *Discriminator     `json:"discriminator,omitempty"`
	Description   string             `json:"description,omitempty"`
	Minimum       *float64           `json:"minimum,omitempty"`
	Maximum       *float64           `json:"maximum,omitempty"`
	MinLength     *int               `json:"minLength,omitempty"`
	MaxLength     *int               `json:"maxLength,omitempty"`
	Pattern       string             `json:"pattern,omitempty"`
	Enum          []string           `json:"enum,omitempty"`
	Items         *Schema            `json:"items,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for Schema to handle the type field correctly.
// If Types is set, it marshals as an array. If Type is set, it marshals as a string.
func (s *Schema) MarshalJSON() ([]byte, error) {
	type Alias Schema
	aux := &struct {
		Type interface{} `json:"type,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if len(s.Types) > 0 {
		aux.Type = s.Types
	} else if s.Type != "" {
		aux.Type = s.Type
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling for Schema to handle the type field correctly.
func (s *Schema) UnmarshalJSON(data []byte) error {
	type Alias Schema
	aux := &struct {
		Type interface{} `json:"type"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Handle the type field based on its actual type
	if aux.Type != nil {
		switch v := aux.Type.(type) {
		case string:
			s.Type = v
		case []interface{}:
			s.Types = make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					s.Types[i] = str
				}
			}
		}
	}

	return nil
}

// Discriminator represents an OpenAPI discriminator object for polymorphic schemas.
type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}

// SecurityScheme represents an OpenAPI security scheme object defining authentication methods.
type SecurityScheme struct {
	Type   string `json:"type"`
	In     string `json:"in,omitempty"`
	Name   string `json:"name,omitempty"`
	Scheme string `json:"scheme,omitempty"`
}

// OpenAPIOption allows callers to tweak the generated specification.
type OpenAPIOption func(*OpenAPISpec)

// WithRouteFilter lets callers provide a predicate deciding whether a given
// RouteInfo should be included in the generated spec. Returning false skips
// the route. Passing this option replaces the default filter (which currently
// removes documentation-serving endpoints).
func WithRouteFilter(f func(*RouteInfo) bool) OpenAPIOption {
	return func(spec *OpenAPISpec) {
		spec.routeFilter = f
	}
}

// WithTitle sets the spec title.
func WithTitle(title string) OpenAPIOption {
	return func(spec *OpenAPISpec) { spec.Info.Title = title }
}

// WithVersion sets the spec version.
func WithVersion(version string) OpenAPIOption {
	return func(spec *OpenAPISpec) { spec.Info.Version = version }
}
