package generator

import (
	"fmt"
	"go/ast"
	"strings"
	"sync"
)

// Generator generates OpenAPI specs from Go code
type Generator struct {
	extractor       *Extractor
	routeDetector   *RouteDetector
	validatorMapper *ValidatorMapper
	spec            *OpenAPISpec

	mu sync.RWMutex
	// Maps for lookups
	typeMap      map[string]ExtractedType
	handlerMap   map[string]ExtractedHandler
	parsedFiles  map[string]*ast.File // filepath -> AST
	filePackages map[string]string    // filepath -> package name
}

// New creates a new generator
func New(title, version string) *Generator {
	return &Generator{
		extractor:       NewExtractor(),
		routeDetector:   NewRouteDetector(),
		validatorMapper: NewValidatorMapper(),
		spec: &OpenAPISpec{
			OpenAPI: "3.1.0",
			Info: Info{
				Title:   title,
				Version: version,
			},
			Paths: make(map[string]*PathItem),
			Components: &Components{
				Schemas:         make(map[string]*Schema),
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		},
		typeMap:      make(map[string]ExtractedType),
		handlerMap:   make(map[string]ExtractedHandler),
		parsedFiles:  make(map[string]*ast.File),
		filePackages: make(map[string]string),
	}
}

// RegisterCustomValidator registers a custom validator
func (g *Generator) RegisterCustomValidator(name, description string) {
	g.validatorMapper.RegisterCustomValidator(name, description)
}

// ParseDirectories parses Go source directories
func (g *Generator) ParseDirectories(dirs []string) error {
	for _, dir := range dirs {
		if err := g.extractor.ParseDirectory(dir); err != nil {
			return fmt.Errorf("failed to parse directory %s: %w", dir, err)
		}
	}

	// Extract types and handlers
	types := g.extractor.ExtractTypes()
	for _, t := range types {
		g.setType(t.Name, t)
		// Also map with package prefix for cross-package references
		g.setType(t.Package+"."+t.Name, t)
	}

	handlers := g.extractor.ExtractHandlers()
	for _, h := range handlers {
		g.setHandler(h.Name, h)
		// Also map with package prefix
		g.setHandler(h.Package+"."+h.Name, h)
	}

	// Store parsed files
	for filePath, file := range g.extractor.GetFiles() {
		g.parsedFiles[filePath] = file
		if file.Name != nil {
			g.filePackages[filePath] = file.Name.Name
		}
	}

	return nil
}

// ParseRoutes parses route registration files
func (g *Generator) ParseRoutes(files []string) error {
	for _, file := range files {
		routes, err := g.routeDetector.DetectRoutesFromFile(file)
		if err != nil {
			return fmt.Errorf("failed to parse routes from %s: %w", file, err)
		}

		for _, route := range routes {
			if err := g.addRoute(route); err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to add route %s %s: %v\n", route.Method, route.Path, err)
			}
		}
	}

	return nil
}

// Generate produces the final OpenAPI spec
func (g *Generator) Generate() *OpenAPISpec {
	// Process handler union types first
	g.ProcessHandlerUnionTypes()

	// Generate schemas for all referenced types
	for _, t := range g.typeMap {
		if !strings.Contains(t.Name, ".") { // Skip duplicates with package prefix
			g.generateSchema(t)
		}
	}

	// Generate tags definitions
	g.generateTags()

	// Generate security schemes
	g.generateSecuritySchemes()

	return g.spec
}

// GetHandlers returns the handler map for debugging
func (g *Generator) GetHandlers() map[string]ExtractedHandler {
	g.mu.RLock()
	defer g.mu.RUnlock()
	// Return a shallow copy to avoid race conditions on caller modification
	copyMap := make(map[string]ExtractedHandler, len(g.handlerMap))
	for k, v := range g.handlerMap {
		copyMap[k] = v
	}
	return copyMap
}

// concurrency helpers
func (g *Generator) getHandler(name string) (ExtractedHandler, bool) {
	g.mu.RLock()
	h, ok := g.handlerMap[name]
	g.mu.RUnlock()
	return h, ok
}

func (g *Generator) setHandler(name string, h ExtractedHandler) {
	g.mu.Lock()
	g.handlerMap[name] = h
	g.mu.Unlock()
}

func (g *Generator) getType(name string) (ExtractedType, bool) {
	g.mu.RLock()
	t, ok := g.typeMap[name]
	g.mu.RUnlock()
	return t, ok
}

func (g *Generator) setType(name string, t ExtractedType) {
	g.mu.Lock()
	g.typeMap[name] = t
	g.mu.Unlock()
}

// generateTags moved to tags.go (no functional change)

// generateSecuritySchemes moved to security.go (no functional change)

// convertSecurityRequirements moved to security.go (no functional change)

// addRoute adds a route to the OpenAPI spec
func (g *Generator) addRoute(route ExtractedRoute) error {
	handler, ok := g.getHandler(route.HandlerName)
	if !ok {
		// Try with package prefix removed
		parts := strings.Split(route.HandlerName, ".")
		if len(parts) > 1 {
			handler, ok = g.getHandler(parts[len(parts)-1])
		}
		if !ok {
			fmt.Printf("Warning: failed to add route %s %s: handler %s not found\n", route.Method, route.Path, route.HandlerName)
			return nil
		}
	}

	// Get or create path item
	pathItem, ok := g.spec.Paths[route.Path]
	if !ok {
		pathItem = &PathItem{}
		g.spec.Paths[route.Path] = pathItem
	}

	// Extract path parameters
	pathParams := ExtractPathParameters(route.Path)

	// Create operation
	operation := &Operation{
		OperationID: handler.Name,
		Description: handler.Description,
		Responses:   make(map[string]*Response),
		Tags:        route.Tags,
	}

	// Add security requirements
	if len(route.Security) > 0 {
		operation.Security = g.convertSecurityRequirements(route.Security)
	}

	// Add path parameters
	for _, param := range pathParams {
		operation.Parameters = append(operation.Parameters, Parameter{
			Name:     param,
			In:       "path",
			Required: true,
			Schema: &Schema{
				Type: "string",
			},
		})
	}

	// Determine method if not specified
	method := route.Method
	if method == "" {
		method = InferMethodFromHandler(handler.Name)
	}

	// Handle request based on method
	// First check if request is a union type
	if unionInfo := DetectUnionType(handler.RequestType); unionInfo.IsUnion {
		// Process union types
		g.ProcessUnionRequestResponse(handler)

		// For union requests, create a reference to the union type itself
		// The union type name needs to be registered as a schema
		unionTypeName := g.normalizeTypeName(handler.RequestType)

		switch method {
		case "POST", "PUT", "PATCH":
			operation.RequestBody = &RequestBody{
				Required: true,
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/" + unionTypeName,
						},
					},
				},
			}
		}
		// GET/DELETE don't typically use union types for query params
	} else if requestType := g.findType(handler.RequestType); requestType != nil {
		switch method {
		case "GET", "DELETE":
			// Add query parameters from request struct
			pathParamNames := make(map[string]bool)
			for _, param := range pathParams {
				pathParamNames[param] = true
			}

			for _, field := range requestType.Fields {
				// Determine OpenAPI tag info
				tagInfo := parseOpenAPITag(field.OpenAPITag)

				// Determine parameter "in" location
				paramIn := tagInfo.In
				if paramIn == "" {
					// If no openapi:in specified and field has json tag, it's body, not query
					if field.JSONTag != "" && field.JSONTag != "-" {
						continue // skip body fields for GET/DELETE
					}
					paramIn = "query" // default for GET/DELETE when no json tag
				}

				// Skip if param location not query/header
				if paramIn != "query" && paramIn != "header" {
					continue
				}

				// Determine parameter name
				paramName := tagInfo.Name
				if paramName == "" {
					paramName = field.JSONTag
				}
				if paramName == "" || paramName == "-" {
					continue // skip unexported
				}

				// Avoid duplicating path params
				if pathParamNames[paramName] {
					continue
				}

				schema := g.fieldToSchema(field, requestType.Name)
				param := Parameter{
					Name:        paramName,
					In:          paramIn,
					Required:    IsRequired(field.ValidateTags),
					Description: field.Description,
					Schema:      schema,
				}
				operation.Parameters = append(operation.Parameters, param)
			}
		case "POST", "PUT", "PATCH":
			// Check for query/header parameters first
			pathParamNames := make(map[string]bool)
			for _, param := range pathParams {
				pathParamNames[param] = true
			}

			for _, field := range requestType.Fields {
				// Determine OpenAPI tag info
				tagInfo := parseOpenAPITag(field.OpenAPITag)

				// Only process fields explicitly marked as query or header
				if tagInfo.In != "query" && tagInfo.In != "header" {
					continue
				}

				// Determine parameter name
				paramName := tagInfo.Name
				if paramName == "" {
					paramName = field.JSONTag
				}
				if paramName == "" || paramName == "-" {
					continue // skip unexported
				}

				// Avoid duplicating path params
				if pathParamNames[paramName] {
					continue
				}

				schema := g.fieldToSchema(field, requestType.Name)
				param := Parameter{
					Name:        paramName,
					In:          tagInfo.In,
					Required:    IsRequired(field.ValidateTags),
					Description: field.Description,
					Schema:      schema,
				}
				operation.Parameters = append(operation.Parameters, param)
			}

			// Use request struct as body
			operation.RequestBody = &RequestBody{
				Required: true,
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/" + requestType.Name,
						},
					},
				},
			}
			// Ensure schema is generated
			g.generateSchema(*requestType)
		}
	}

	// Handle response
	// First check if response is a union type
	if unionInfo := DetectUnionType(handler.ResponseType); unionInfo.IsUnion {
		// Process union types
		g.ProcessUnionRequestResponse(handler)

		// For union responses, create a reference to the union type itself
		unionTypeName := g.normalizeTypeName(handler.ResponseType)

		operation.Responses["200"] = &Response{
			Description: "Successful response",
			Content: map[string]*MediaType{
				"application/json": {
					Schema: &Schema{
						Ref: "#/components/schemas/" + unionTypeName,
					},
				},
			},
		}
	} else if responseType := g.findType(handler.ResponseType); responseType != nil {
		operation.Responses["200"] = &Response{
			Description: "Successful response",
			Content: map[string]*MediaType{
				"application/json": {
					Schema: &Schema{
						Ref: "#/components/schemas/" + responseType.Name,
					},
				},
			},
		}
		// Ensure schema is generated
		g.generateSchema(*responseType)
	} else {
		// Simple type response or no response
		operation.Responses["200"] = &Response{
			Description: "Successful response",
		}
	}

	// Add error responses
	operation.Responses["400"] = &Response{
		Description: "Bad request",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"error": {
							Type:        "string",
							Description: "Error message",
						},
					},
				},
			},
		},
	}

	operation.Responses["500"] = &Response{
		Description: "Internal server error",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"error": {
							Type:        "string",
							Description: "Error message",
						},
					},
				},
			},
		},
	}

	// Assign operation to method
	switch method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	}

	return nil
}

// findType finds a type by name, handling package prefixes
func (g *Generator) findType(typeName string) *ExtractedType {
	// Remove pointer prefix
	typeName = strings.TrimPrefix(typeName, "*")

	// Try direct lookup
	if t, ok := g.getType(typeName); ok {
		return &t
	}

	// Try without package prefix
	parts := strings.Split(typeName, ".")
	if len(parts) > 1 {
		if t, ok := g.getType(parts[len(parts)-1]); ok {
			return &t
		}
	}

	return nil
}

// generateSchema generates a schema for a type
func (g *Generator) generateSchema(t ExtractedType) {
	if _, exists := g.spec.Components.Schemas[t.Name]; exists {
		return // already generated
	}

	// Handle union type aliases first
	if t.IsUnionAlias {
		unionSchema := g.generateUnionAliasSchema(t)
		g.spec.Components.Schemas[t.Name] = unionSchema
		return
	}

	// Handle simple type aliases (e.g., type MyString string)
	if t.IsTypeAlias && t.BaseType != "" {
		schema := g.generateTypeAliasSchema(t)
		g.spec.Components.Schemas[t.Name] = schema
		return
	}

	schema := &Schema{
		Type:        "object",
		Description: t.Description,
		Properties:  make(map[string]*Schema),
	}

	var required []string

	// Check if this might be a union options type (all fields are pointers with no JSON tags)
	isUnionOptions := g.isUnionOptionsType(t)

	// First, process embedded types
	for _, embeddedTypeName := range t.EmbeddedTypes {
		if embeddedType := g.findType(embeddedTypeName); embeddedType != nil {
			// Recursively ensure the embedded type's schema is generated
			g.generateSchema(*embeddedType)

			// Add fields from embedded type
			for _, field := range embeddedType.Fields {
				// Check if this field has an openapi tag that places it outside the body
				tagInfo := parseOpenAPITag(field.OpenAPITag)
				if tagInfo.In == "query" || tagInfo.In == "header" || tagInfo.In == "path" {
					// Skip fields that are not part of the request body
					continue
				}

				// For union options types, include fields even without JSON tags
				// For regular types, skip fields without JSON tags
				if !isUnionOptions && (field.JSONTag == "" || field.JSONTag == "-") {
					continue
				}

				fieldName := field.JSONTag
				if fieldName == "" {
					// Use the field name directly for union options
					fieldName = strings.ToLower(field.Name)
				}

				fieldSchema := g.fieldToSchema(field, embeddedType.Name)
				schema.Properties[fieldName] = fieldSchema

				if IsRequired(field.ValidateTags) {
					required = append(required, fieldName)
				}
			}
		}
	}

	// Then process regular fields
	for _, field := range t.Fields {
		// Check if this field has an openapi tag that places it outside the body
		tagInfo := parseOpenAPITag(field.OpenAPITag)
		if tagInfo.In == "query" || tagInfo.In == "header" || tagInfo.In == "path" {
			// Skip fields that are not part of the request body
			continue
		}

		// For union options types, include fields even without JSON tags
		// For regular types, skip fields without JSON tags
		if !isUnionOptions && (field.JSONTag == "" || field.JSONTag == "-") {
			continue
		}

		fieldName := field.JSONTag
		if fieldName == "" {
			// Use the field name directly for union options
			fieldName = strings.ToLower(field.Name)
		}

		fieldSchema := g.fieldToSchema(field, t.Name)
		schema.Properties[fieldName] = fieldSchema

		if IsRequired(field.ValidateTags) {
			required = append(required, fieldName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	g.spec.Components.Schemas[t.Name] = schema
}

// fieldToSchema converts a field to a schema
func (g *Generator) fieldToSchema(field ExtractedField, parentType string) *Schema {
	// Check if this is a union type first
	if unionInfo := DetectUnionType(field.Type); unionInfo.IsUnion {
		return g.GenerateUnionSchema(unionInfo, field.Name, field)
	}

	schema := &Schema{
		Description: field.Description,
	}

	// Handle pointer types
	if field.IsPointer {
		schema.Nullable = true
	}

	// Determine base type
	fieldType := field.Type
	isArray := false
	isMap := false

	if strings.HasPrefix(fieldType, "[]") {
		isArray = true
		fieldType = strings.TrimPrefix(fieldType, "[]")
	} else if strings.HasPrefix(fieldType, "map[") {
		isMap = true
		// Extract value type from map[K]V
		if idx := strings.LastIndex(fieldType, "]"); idx > 0 {
			fieldType = fieldType[idx+1:]
		}
	}

	// Set base type
	switch fieldType {
	case "string":
		schema.Type = "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		schema.Type = "integer"
	case "float32", "float64":
		schema.Type = "number"
	case "bool":
		schema.Type = "boolean"
	case "time.Time":
		schema.Type = "string"
		schema.Format = "date-time"
	case "[]byte":
		schema.Type = "string"
		schema.Format = "byte"
	default:
		// Check if it's a known type
		if refType := g.findType(fieldType); refType != nil {
			// Ensure schema is generated
			g.generateSchema(*refType)

			if isArray {
				schema.Type = "array"
				schema.Items = &Schema{
					Ref: "#/components/schemas/" + refType.Name,
				}
			} else {
				// Reference to another schema
				return &Schema{
					Ref:         "#/components/schemas/" + refType.Name,
					Description: field.Description,
					Nullable:    field.IsPointer,
				}
			}
		} else {
			// Unknown type, default to object
			schema.Type = "object"
		}
	}

	// Handle array/map wrappers
	if isArray && schema.Type != "array" {
		itemSchema := &Schema{
			Type: schema.Type,
		}
		if schema.Format != "" {
			itemSchema.Format = schema.Format
			schema.Format = ""
		}
		schema.Type = "array"
		schema.Items = itemSchema
	} else if isMap {
		schema.Type = "object"
		schema.AdditionalProperties = true
	}

	// Apply validator constraints
	if field.ValidateTags != "" {
		// Handle dive for arrays
		if isArray && strings.Contains(field.ValidateTags, "dive") {
			// Apply validation to items
			if schema.Items != nil {
				if err := g.validatorMapper.MapValidatorTags(field.ValidateTags, schema.Items, fieldType); err != nil {
					// Log warning but continue - invalid validator tags shouldn't break generation
					if schema.Description == "" {
						schema.Description = fmt.Sprintf("Warning: %v", err)
					} else {
						schema.Description += fmt.Sprintf(" (Warning: %v)", err)
					}
				}
			}
		} else {
			if err := g.validatorMapper.MapValidatorTags(field.ValidateTags, schema, field.Type); err != nil {
				// Log warning but continue - invalid validator tags shouldn't break generation
				if schema.Description == "" {
					schema.Description = fmt.Sprintf("Warning: %v", err)
				} else {
					schema.Description += fmt.Sprintf(" (Warning: %v)", err)
				}
			}
		}
	}

	return schema
}

// generateUnionAliasSchema generates a schema for a union type alias
func (g *Generator) generateUnionAliasSchema(t ExtractedType) *Schema {
	// Create a dummy field to reuse the existing union schema generation logic
	dummyField := ExtractedField{
		Name:        t.Name,
		Type:        t.AliasedType,
		Description: t.Description,
	}

	// Generate the union schema using the appropriate method
	unionSchema := g.GenerateUnionSchema(t.UnionInfo, t.Name, dummyField)

	// Ensure all referenced types are processed
	for _, typeName := range t.UnionInfo.UnionTypes {
		g.ensureTypeProcessed(typeName)
	}

	return unionSchema
}

// generateTypeAliasSchema generates a schema for a simple type alias
func (g *Generator) generateTypeAliasSchema(t ExtractedType) *Schema {
	schema := &Schema{
		Description: t.Description,
	}

	// Map Go base types to OpenAPI types
	switch t.BaseType {
	case "string":
		schema.Type = "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		schema.Type = "integer"
	case "float32", "float64":
		schema.Type = "number"
	case "bool":
		schema.Type = "boolean"
	default:
		// Default to string if unknown
		schema.Type = "string"
	}

	// If we have enum values, add them
	if len(t.EnumValues) > 0 {
		schema.Enum = make([]interface{}, len(t.EnumValues))
		for i, v := range t.EnumValues {
			schema.Enum[i] = v
		}
	}

	return schema
}

// isUnionOptionsType checks if a type is likely a union options type
// (all fields are pointers with no JSON tags)
func (g *Generator) isUnionOptionsType(t ExtractedType) bool {
	// If the type has no direct fields but has embedded types,
	// check the embedded types
	if len(t.Fields) == 0 && len(t.EmbeddedTypes) > 0 {
		// Check if any embedded type is a union options type
		for _, embeddedTypeName := range t.EmbeddedTypes {
			if embeddedType := g.findType(embeddedTypeName); embeddedType != nil {
				if g.isUnionOptionsType(*embeddedType) {
					return true
				}
			}
		}
		return false
	}

	if len(t.Fields) == 0 {
		return false
	}

	// Check if all fields are pointers without JSON tags
	for _, field := range t.Fields {
		if !field.IsPointer || field.JSONTag != "" {
			return false
		}
	}

	return true
}

// GetExtractedTypes returns all extracted types
func (g *Generator) GetExtractedTypes() []ExtractedType {
	types := make([]ExtractedType, 0, len(g.typeMap))
	seen := make(map[string]bool)

	for _, t := range g.typeMap {
		// Avoid duplicates (same type might be mapped with and without package prefix)
		key := t.Package + "." + t.Name
		if !seen[key] {
			types = append(types, t)
			seen[key] = true
		}
	}

	return types
}
