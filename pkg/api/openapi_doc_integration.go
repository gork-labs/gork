package api

import "strings"

// SchemaFieldSuffix represents the various field suffixes used in contextual schema naming.
type SchemaFieldSuffix string

// String returns the string representation of the schema field suffix.
func (s SchemaFieldSuffix) String() string {
	return string(s)
}

const (
	// SchemaSuffixBody represents request/response body schemas.
	SchemaSuffixBody SchemaFieldSuffix = "Body"
	// SchemaSuffixHeaders represents request/response header schemas.
	SchemaSuffixHeaders SchemaFieldSuffix = "Headers"
	// SchemaSuffixQuery represents request query parameter schemas.
	SchemaSuffixQuery SchemaFieldSuffix = "Query"
	// SchemaSuffixPath represents request path parameter schemas.
	SchemaSuffixPath SchemaFieldSuffix = "Path"
	// SchemaSuffixCookies represents request/response cookie schemas.
	SchemaSuffixCookies SchemaFieldSuffix = "Cookies"
	// SchemaSuffixResponse represents response schemas.
	SchemaSuffixResponse SchemaFieldSuffix = "Response"
)

// Integration of AST documentation into the runtime-generated OpenAPI spec.

// GenerateOpenAPIWithDocs combines route information from the given registry
// with documentation parsed by DocExtractor to enrich operation and schema
// descriptions. The function delegates the core generation work to
// GenerateOpenAPI and then post-processes the specification.
func GenerateOpenAPIWithDocs(reg *RouteRegistry, extractor *DocExtractor, opts ...OpenAPIOption) *OpenAPISpec {
	spec := GenerateOpenAPI(reg, opts...)
	if extractor == nil {
		return spec
	}

	// Enrich component schemas first so that operations using $ref automatically
	// pick up descriptions.
	enrichComponentSchemas(spec, extractor)

	// Update path operations.
	enrichPathOperations(spec, extractor)

	return spec
}

func enrichComponentSchemas(spec *OpenAPISpec, extractor *DocExtractor) {
	for name, schema := range spec.Components.Schemas {
		enrichSchemaWithTypeDoc(schema, name, extractor)
	}
}

func enrichSchemaWithTypeDoc(schema *Schema, typeName string, extractor *DocExtractor) {
	// First try the schema name directly
	doc := extractor.ExtractTypeDoc(typeName)

	if doc.Description != "" {
		schema.Description = doc.Description
	}
	enrichSchemaPropertiesWithDocs(schema, doc)

	// Check if we still have properties without descriptions that might come from embedded types
	// For contextual schema names, prioritize the matching request type
	if isContextualSchemaName(typeName) {
		enrichFromContextualRequestType(schema, typeName, extractor)
	} else {
		enrichFromEmbeddedTypes(schema, extractor)
	}
}

func enrichSchemaPropertiesWithDocs(schema *Schema, doc Documentation) {
	if len(doc.Fields) == 0 || schema.Properties == nil {
		return
	}
	for propName, propSchema := range schema.Properties {
		if fd, ok := doc.Fields[propName]; ok {
			if propSchema.Description == "" {
				propSchema.Description = fd.Description
			}
		}
	}
}

// TypeDocExtractor defines the interface needed for enrichFromEmbeddedTypes.
type TypeDocExtractor interface {
	GetAllTypeNames() []string
	ExtractTypeDoc(typeName string) Documentation
}

func enrichFromEmbeddedTypes(schema *Schema, extractor TypeDocExtractor) {
	if schema == nil || schema.Properties == nil {
		return
	}

	propsNeedingDocs := findPropertiesNeedingDocs(schema)
	if len(propsNeedingDocs) == 0 {
		return
	}

	enrichPropertiesFromTypes(propsNeedingDocs, extractor)
}

// findPropertiesNeedingDocs finds properties that still don't have descriptions.
func findPropertiesNeedingDocs(schema *Schema) map[string]*Schema {
	propsNeedingDocs := make(map[string]*Schema)
	for propName, propSchema := range schema.Properties {
		if propSchema.Description == "" {
			propsNeedingDocs[propName] = propSchema
		}
	}
	return propsNeedingDocs
}

// enrichPropertiesFromTypes tries to enrich properties from documented types.
func enrichPropertiesFromTypes(propsNeedingDocs map[string]*Schema, extractor TypeDocExtractor) {
	enrichPropertiesWithPreferredType(propsNeedingDocs, "", extractor)
}

// enrichPropertiesWithPreferredType tries to enrich properties, prioritizing a preferred type.
func enrichPropertiesWithPreferredType(propsNeedingDocs map[string]*Schema, preferredType string, extractor TypeDocExtractor) {
	// Try preferred type first if specified
	if tryPreferredTypeEnrichment(propsNeedingDocs, preferredType, extractor) {
		return // All properties documented
	}

	// Try remaining types for any undocumented properties
	tryRemainingTypesEnrichment(propsNeedingDocs, preferredType, extractor)
}

// tryPreferredTypeEnrichment attempts to enrich properties from the preferred type.
// Returns true if all properties were successfully documented.
func tryPreferredTypeEnrichment(propsNeedingDocs map[string]*Schema, preferredType string, extractor TypeDocExtractor) bool {
	if preferredType == "" {
		return false
	}

	preferredDoc := extractor.ExtractTypeDoc(preferredType)
	if len(preferredDoc.Fields) == 0 {
		return false
	}

	if !tryEnrichFromType(propsNeedingDocs, preferredDoc) {
		return false
	}

	// Remove successfully documented properties
	removeDocumentedProperties(propsNeedingDocs, preferredDoc)
	return len(propsNeedingDocs) == 0
}

// removeDocumentedProperties removes properties that were successfully documented.
func removeDocumentedProperties(propsNeedingDocs map[string]*Schema, typeDoc Documentation) {
	for propName := range propsNeedingDocs {
		if _, hasDoc := typeDoc.Fields[propName]; hasDoc {
			delete(propsNeedingDocs, propName)
		}
	}
}

// tryRemainingTypesEnrichment tries to enrich remaining properties from other types.
func tryRemainingTypesEnrichment(propsNeedingDocs map[string]*Schema, preferredType string, extractor TypeDocExtractor) {
	allTypes := extractor.GetAllTypeNames()
	for _, typeName := range allTypes {
		if typeName == preferredType {
			continue // Already tried this one
		}

		typeDoc := extractor.ExtractTypeDoc(typeName)
		if len(typeDoc.Fields) == 0 {
			continue
		}

		if tryEnrichFromType(propsNeedingDocs, typeDoc) {
			break // Found documentation, stop looking
		}
	}
}

// tryEnrichFromType attempts to enrich properties from a single type's documentation.
func tryEnrichFromType(propsNeedingDocs map[string]*Schema, typeDoc Documentation) bool {
	matches := countFieldMatches(propsNeedingDocs, typeDoc)
	if matches == 0 {
		return false
	}

	// Apply documentation to matching fields
	for propName, propSchema := range propsNeedingDocs {
		if fieldDoc, hasDoc := typeDoc.Fields[propName]; hasDoc {
			propSchema.Description = fieldDoc.Description
		}
	}
	return true
}

// countFieldMatches counts how many properties can be documented by this type.
func countFieldMatches(propsNeedingDocs map[string]*Schema, typeDoc Documentation) int {
	matches := 0
	for propName := range propsNeedingDocs {
		if _, hasDoc := typeDoc.Fields[propName]; hasDoc {
			matches++
		}
	}
	return matches
}

func enrichPathOperations(spec *OpenAPISpec, extractor *DocExtractor) {
	for _, item := range spec.Paths {
		updateOperationWithDocs(item.Get, extractor)
		updateOperationWithDocs(item.Post, extractor)
		updateOperationWithDocs(item.Put, extractor)
		updateOperationWithDocs(item.Patch, extractor)
		updateOperationWithDocs(item.Delete, extractor)
	}
}

func updateOperationWithDocs(op *Operation, extractor *DocExtractor) {
	if op == nil || extractor == nil {
		return
	}
	doc := extractor.ExtractFunctionDoc(op.OperationID)
	if doc.Description != "" {
		op.Description = doc.Description
	}

	// Enhance parameters with documentation
	enrichParametersWithDocs(op, extractor)
}

// enrichParametersWithDocs adds field documentation to operation parameters.
func enrichParametersWithDocs(op *Operation, extractor *DocExtractor) {
	if op == nil || extractor == nil || len(op.Parameters) == 0 {
		return
	}

	// Try to find request type documentation by looking for types that match the handler pattern
	// Convention: handlers like "GetUser" typically have request types like "GetUserRequest"
	requestTypeName := op.OperationID + "Request"
	requestDoc := extractor.ExtractTypeDoc(requestTypeName)

	if len(requestDoc.Fields) == 0 {
		return // No field documentation available
	}

	// Enhance each parameter with field documentation
	for i := range op.Parameters {
		param := &op.Parameters[i]
		if fieldDoc, hasDoc := requestDoc.Fields[param.Name]; hasDoc {
			param.Description = fieldDoc.Description
		}
	}
}

// EnhanceOpenAPISpecWithDocs enriches an already generated specification with
// documentation extracted from source code. It can be used when the spec was
// produced by a separate process (e.g. a runtime export) and therefore we no
// longer have access to the RouteRegistry.
func EnhanceOpenAPISpecWithDocs(spec *OpenAPISpec, extractor *DocExtractor) {
	if spec == nil || extractor == nil {
		return
	}

	// Component schemas
	if spec.Components != nil {
		enrichComponentSchemas(spec, extractor)
	}

	// Paths & operations
	enrichPathOperations(spec, extractor)
}

// isContextualSchemaName determines if a schema name was generated using contextual naming.
// This includes request body schemas (ends with "Body"), response schemas, and potentially
// future contextual schemas for headers, query parameters, etc.
func isContextualSchemaName(schemaName string) bool {
	return strings.HasSuffix(schemaName, SchemaSuffixBody.String()) ||
		strings.HasSuffix(schemaName, SchemaSuffixHeaders.String()) ||
		strings.HasSuffix(schemaName, SchemaSuffixQuery.String()) ||
		strings.HasSuffix(schemaName, SchemaSuffixPath.String()) ||
		strings.HasSuffix(schemaName, SchemaSuffixCookies.String()) ||
		strings.HasSuffix(schemaName, SchemaSuffixResponse.String())
}

// enrichFromContextualRequestType enriches schema properties from the specific request type
// that matches the contextual schema name. This ensures we look at the right request type first.
func enrichFromContextualRequestType(schema *Schema, schemaName string, extractor TypeDocExtractor) {
	if schema == nil || schema.Properties == nil {
		return
	}

	propsNeedingDocs := findPropertiesNeedingDocs(schema)
	if len(propsNeedingDocs) == 0 {
		return
	}

	// Get the preferred request type name for this contextual schema
	preferredRequestType := getRequestTypeNameFromContextualSchema(schemaName)

	// Use a modified approach that prioritizes the preferred request type
	enrichPropertiesWithPreferredType(propsNeedingDocs, preferredRequestType, extractor)
}

// getRequestTypeNameFromContextualSchema maps a contextual schema name to its request type.
// Examples: "UpdateUserBody" -> "UpdateUserRequest", "CreateUserHeaders" -> "CreateUserRequest".
func getRequestTypeNameFromContextualSchema(schemaName string) string {
	if strings.HasSuffix(schemaName, SchemaSuffixBody.String()) {
		baseName := strings.TrimSuffix(schemaName, SchemaSuffixBody.String())
		return baseName + "Request"
	}

	if strings.HasSuffix(schemaName, SchemaSuffixHeaders.String()) {
		baseName := strings.TrimSuffix(schemaName, SchemaSuffixHeaders.String())
		return baseName + "Request"
	}

	if strings.HasSuffix(schemaName, SchemaSuffixQuery.String()) {
		baseName := strings.TrimSuffix(schemaName, SchemaSuffixQuery.String())
		return baseName + "Request"
	}

	if strings.HasSuffix(schemaName, SchemaSuffixPath.String()) {
		baseName := strings.TrimSuffix(schemaName, SchemaSuffixPath.String())
		return baseName + "Request"
	}

	if strings.HasSuffix(schemaName, SchemaSuffixCookies.String()) {
		baseName := strings.TrimSuffix(schemaName, SchemaSuffixCookies.String())
		return baseName + "Request"
	}

	if strings.HasSuffix(schemaName, SchemaSuffixResponse.String()) {
		return schemaName // Response types should already be named correctly
	}

	return ""
}
