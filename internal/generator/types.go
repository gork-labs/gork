package generator

import "encoding/json"

// OpenAPISpec represents the OpenAPI 3.1.0 specification
type OpenAPISpec struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Paths      map[string]*PathItem  `json:"paths,omitempty"`
	Components *Components           `json:"components,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty"`
}

// Info represents the info object
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// Tag represents a tag for grouping operations
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SecurityScheme represents a security scheme
type SecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`         // For apiKey
	In           string `json:"in,omitempty"`           // For apiKey
	Scheme       string `json:"scheme,omitempty"`       // For http
	BearerFormat string `json:"bearerFormat,omitempty"` // For http bearer
}

// PathItem represents operations for a specific path
type PathItem struct {
	Get        *Operation  `json:"get,omitempty"`
	Post       *Operation  `json:"post,omitempty"`
	Put        *Operation  `json:"put,omitempty"`
	Delete     *Operation  `json:"delete,omitempty"`
	Patch      *Operation  `json:"patch,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

// Operation represents an API operation
type Operation struct {
	OperationID string                   `json:"operationId,omitempty"`
	Summary     string                   `json:"summary,omitempty"`
	Description string                   `json:"description,omitempty"`
	Parameters  []Parameter              `json:"parameters,omitempty"`
	RequestBody *RequestBody             `json:"requestBody,omitempty"`
	Responses   map[string]*Response     `json:"responses"`
	Tags        []string                 `json:"tags,omitempty"`
	Security    []map[string][]string    `json:"security,omitempty"`
}

// Parameter represents a parameter in an operation
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

// Response represents an API response
type Response struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
}

// MediaType represents a media type
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Components holds reusable objects
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// Schema represents a JSON Schema
type Schema struct {
	Ref                  string                 `json:"$ref,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Properties           map[string]*Schema     `json:"properties,omitempty"`
	Items                *Schema                `json:"items,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Example              interface{}            `json:"example,omitempty"`
	Minimum              *float64               `json:"minimum,omitempty"`
	Maximum              *float64               `json:"maximum,omitempty"`
	ExclusiveMinimum     bool                   `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     bool                   `json:"exclusiveMaximum,omitempty"`
	MinLength            *int                   `json:"minLength,omitempty"`
	MaxLength            *int                   `json:"maxLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	MinItems             *int                   `json:"minItems,omitempty"`
	MaxItems             *int                   `json:"maxItems,omitempty"`
	UniqueItems          bool                   `json:"uniqueItems,omitempty"`
	MinProperties        *int                   `json:"minProperties,omitempty"`
	MaxProperties        *int                   `json:"maxProperties,omitempty"`
	Nullable             bool                   `json:"nullable,omitempty"`
	ReadOnly             bool                   `json:"readOnly,omitempty"`
	WriteOnly            bool                   `json:"writeOnly,omitempty"`
	Deprecated           bool                   `json:"deprecated,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	AllOf                []*Schema              `json:"allOf,omitempty"`
	OneOf                []*Schema              `json:"oneOf,omitempty"`
	AnyOf                []*Schema              `json:"anyOf,omitempty"`
	Discriminator        *Discriminator         `json:"discriminator,omitempty"`
}

// Discriminator represents the discriminator object for polymorphism
type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}

// ExtractedType represents a Go type extracted from source
type ExtractedType struct {
	Name         string
	Package      string
	Description  string
	Fields       []ExtractedField
	EmbeddedTypes []string  // Names of embedded types
	IsUnionAlias bool      // True if this is a union type alias
	UnionInfo    UnionInfo // Union information if IsUnionAlias is true
	IsTypeAlias  bool      // True if this is any type alias (including non-union)
	AliasedType  string    // The aliased type string
	BaseType     string    // The base type for simple aliases (e.g., "string" for type MyString string)
	EnumValues   []string  // Possible enum values if constants are defined
	TypeDef      string    // For type aliases, the full type definition
	SourceFile   string    // Path to the source file containing this type
}

// ExtractedField represents a struct field
type ExtractedField struct {
	Name         string
	Type         string
	JSONTag      string
	ValidateTags string
	OpenAPITag   string
	Description  string
	IsPointer    bool
}

// ExtractedHandler represents a handler function
type ExtractedHandler struct {
	Name         string
	Package      string
	Description  string
	RequestType  string
	ResponseType string
}

// ExtractedRoute represents a route registration
type ExtractedRoute struct {
	Method      string
	Path        string
	HandlerName string
	Tags        []string
	Security    []SecurityRequirement
}

// MarshalJSON customizes JSON marshaling to omit empty values properly
func (s *Schema) MarshalJSON() ([]byte, error) {
	type schemaAlias Schema
	
	// Create a temporary struct with proper omitempty handling
	temp := struct {
		schemaAlias
		ExclusiveMinimum *bool `json:"exclusiveMinimum,omitempty"`
		ExclusiveMaximum *bool `json:"exclusiveMaximum,omitempty"`
		UniqueItems      *bool `json:"uniqueItems,omitempty"`
		Nullable         *bool `json:"nullable,omitempty"`
		ReadOnly         *bool `json:"readOnly,omitempty"`
		WriteOnly        *bool `json:"writeOnly,omitempty"`
		Deprecated       *bool `json:"deprecated,omitempty"`
	}{
		schemaAlias: schemaAlias(*s),
	}
	
	// Only set boolean fields if they're true
	if s.ExclusiveMinimum {
		b := true
		temp.ExclusiveMinimum = &b
	}
	if s.ExclusiveMaximum {
		b := true
		temp.ExclusiveMaximum = &b
	}
	if s.UniqueItems {
		b := true
		temp.UniqueItems = &b
	}
	if s.Nullable {
		b := true
		temp.Nullable = &b
	}
	if s.ReadOnly {
		b := true
		temp.ReadOnly = &b
	}
	if s.WriteOnly {
		b := true
		temp.WriteOnly = &b
	}
	if s.Deprecated {
		b := true
		temp.Deprecated = &b
	}
	
	return json.Marshal(temp)
}