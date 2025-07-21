package api

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

type Info struct {
	Title   string `json:"title,omitempty" yaml:"title,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty" yaml:"responses,omitempty"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post   *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put    *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
}

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

type Parameter struct {
	Name     string  `json:"name" yaml:"name"`
	In       string  `json:"in" yaml:"in"` // "query", "header", "path", "cookie"
	Required bool    `json:"required" yaml:"required"`
	Schema   *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type RequestBody struct {
	Required bool                 `json:"required,omitempty" yaml:"required,omitempty"`
	Content  map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type Response struct {
	Ref         string               `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type Schema struct {
	// Title provides a human-readable name for the schema. Some documentation UIs
	// (e.g. ReDoc, Swagger UI) display this value in type signatures. We set it
	// automatically from the Go type name where available so that arrays like
	// []UserResponse are shown as array[UserResponse] instead of array[object].
	Title         string             `json:"title,omitempty" yaml:"title,omitempty"`
	Ref           string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type          string             `json:"type,omitempty" yaml:"type,omitempty"`
	Properties    map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required      []string           `json:"required,omitempty" yaml:"required,omitempty"`
	OneOf         []*Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
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

type Discriminator struct {
	PropertyName string            `json:"propertyName" yaml:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

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
