package api

import "encoding/json"

// NOTE: These definitions intentionally keep only the fields we actively
// populate for now. Additional fields can be added without breaking existing
// users as we expand the generator.

// OpenAPISpec represents the root of an OpenAPI 3.1 document.
type OpenAPISpec struct {
	OpenAPI    string               `json:"openapi" yaml:"openapi"`
	Info       Info                 `json:"info" yaml:"info"`
	Paths      map[string]*PathItem `json:"paths" yaml:"paths"`
	Components *Components          `json:"components,omitempty" yaml:"components,omitempty"`

	// routeFilter allows callers to skip specific RouteInfo entries during
	// spec generation. It is internal-only and therefore excluded from JSON
	// and YAML output.
	routeFilter func(*RouteInfo) bool `json:"-" yaml:"-"`
}

// Info represents the OpenAPI info section containing metadata about the API.
type Info struct {
	Title   string `json:"title,omitempty" yaml:"title,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// Components represents the OpenAPI components section containing reusable objects.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty" yaml:"responses,omitempty"`
}

// PathItem represents a path item object containing HTTP operations for a path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post   *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put    *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
}

// Operation represents an OpenAPI operation object describing a single API operation.
type Operation struct {
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]*Response  `json:"responses,omitempty" yaml:"responses,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

// Parameter represents an OpenAPI parameter object describing a single operation parameter.
type Parameter struct {
	Name     string  `json:"name" yaml:"name"`
	In       string  `json:"in" yaml:"in"` // "query", "header", "path", "cookie"
	Required bool    `json:"required" yaml:"required"`
	Schema   *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// RequestBody represents an OpenAPI request body object.
type RequestBody struct {
	Required bool                 `json:"required,omitempty" yaml:"required,omitempty"`
	Content  map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

// MediaType represents an OpenAPI media type object containing schema information.
type MediaType struct {
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// Response represents an OpenAPI response object describing a single response from an API operation.
type Response struct {
	Ref         string               `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

// Schema represents an OpenAPI schema object defining the structure of request/response data.
type Schema struct {
	// Title provides a human-readable name for the schema. Some documentation UIs
	// (e.g. ReDoc, Swagger UI) display this value in type signatures. We set it
	// automatically from the Go type name where available so that arrays like
	// []UserResponse are shown as array[UserResponse] instead of array[object].
	Title         string             `json:"title,omitempty" yaml:"title,omitempty"`
	Ref           string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type          string             `json:"-" yaml:"-"`
	Types         []string           `json:"-" yaml:"-"`
	Properties    map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required      []string           `json:"required,omitempty" yaml:"required,omitempty"`
	OneOf         []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf         []*Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Discriminator *Discriminator     `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
	Description   string             `json:"description,omitempty" yaml:"description,omitempty"`
	Minimum       *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum       *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	MinLength     *int               `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength     *int               `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Pattern       string             `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Enum          []string           `json:"enum,omitempty" yaml:"enum,omitempty"`
	Items         *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
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

// MarshalYAML implements custom YAML marshaling for Schema to handle the type field correctly.
func (s *Schema) MarshalYAML() (interface{}, error) {
	type Alias Schema
	aux := &struct {
		Type interface{} `yaml:"type,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if len(s.Types) > 0 {
		aux.Type = s.Types
	} else if s.Type != "" {
		aux.Type = s.Type
	}

	return aux, nil
}

// UnmarshalYAML implements custom YAML unmarshaling for Schema to handle the type field correctly.
func (s *Schema) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Alias Schema
	aux := &struct {
		Type interface{} `yaml:"type"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := unmarshal(aux); err != nil {
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
	PropertyName string            `json:"propertyName" yaml:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

// SecurityScheme represents an OpenAPI security scheme object defining authentication methods.
type SecurityScheme struct {
	Type   string `json:"type" yaml:"type"`
	In     string `json:"in,omitempty" yaml:"in,omitempty"`
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
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
