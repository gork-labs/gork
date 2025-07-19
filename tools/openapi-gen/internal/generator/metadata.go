package generator

// RouteMetadata stores metadata about routes that can't be extracted from AST alone
type RouteMetadata struct {
	Path     string
	Method   string
	Tags     []string
	Security []SecurityRequirement
}

// SecurityRequirement represents a security requirement
type SecurityRequirement struct {
	Type   string   // basic, bearer, apiKey
	Scopes []string // For OAuth2
}

// MetadataStore stores route metadata
var MetadataStore = make(map[string]*RouteMetadata)

// AddRouteMetadata adds metadata for a route
func AddRouteMetadata(path, method string, tags []string, security []SecurityRequirement) {
	key := method + ":" + path
	MetadataStore[key] = &RouteMetadata{
		Path:     path,
		Method:   method,
		Tags:     tags,
		Security: security,
	}
}

// GetRouteMetadata retrieves metadata for a route
func GetRouteMetadata(path, method string) *RouteMetadata {
	key := method + ":" + path
	return MetadataStore[key]
}

// For demonstration, we'll hardcode the metadata based on the example routes
func init() {
	// User Management
	AddRouteMetadata("/users", "GET", []string{"User Management"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"read:users"}}})
	AddRouteMetadata("/users/create", "POST", []string{"User Management"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"write:users"}}})
	AddRouteMetadata("/users/get", "GET", []string{"User Management"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"read:users"}}})
	AddRouteMetadata("/users/update", "PUT", []string{"User Management"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"write:users"}}})
	AddRouteMetadata("/users/delete", "DELETE", []string{"User Management"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"admin"}}})
	
	// Product Catalog
	AddRouteMetadata("/products", "GET", []string{"Product Catalog", "Public"}, nil)
	AddRouteMetadata("/products/create", "POST", []string{"Product Catalog"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"write:products"}}})
	AddRouteMetadata("/products/get", "GET", []string{"Product Catalog", "Public"}, nil)
	AddRouteMetadata("/products/update", "PUT", []string{"Product Catalog"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"write:products"}}})
	AddRouteMetadata("/products/delete", "DELETE", []string{"Product Catalog"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"admin"}}})
	AddRouteMetadata("/products/images/add", "POST", []string{"Product Catalog"}, []SecurityRequirement{{Type: "bearer", Scopes: []string{"write:products"}}})
	
	// Demo routes
	AddRouteMetadata("/demo/basic", "GET", []string{"Demo"}, []SecurityRequirement{{Type: "basic"}})
	AddRouteMetadata("/demo/apikey", "GET", []string{"Demo"}, []SecurityRequirement{{Type: "apiKey"}})
}