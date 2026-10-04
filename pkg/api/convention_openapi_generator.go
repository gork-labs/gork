package api

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gork-labs/gork/pkg/gorkson"
)

// ConventionOpenAPIGenerator generates OpenAPI specs for Convention Over Configuration handlers.
type ConventionOpenAPIGenerator struct {
	spec      *OpenAPISpec
	extractor *DocExtractor
}

// NewConventionOpenAPIGenerator creates a new convention OpenAPI generator.
func NewConventionOpenAPIGenerator(spec *OpenAPISpec, extractor *DocExtractor) *ConventionOpenAPIGenerator {
	return &ConventionOpenAPIGenerator{
		spec:      spec,
		extractor: extractor,
	}
}

// buildConventionOperation builds an OpenAPI operation for Convention Over Configuration requests.
func (g *ConventionOpenAPIGenerator) buildConventionOperation(route *RouteInfo, components *Components) *Operation {
	operation := &Operation{
		OperationID: route.HandlerName,
		Parameters:  []Parameter{},
		Responses:   map[string]*Response{},
	}

	// Add tags if options are provided
	if route.Options != nil {
		operation.Tags = route.Options.Tags
	}

	// Check if this is a webhook handler
	isWebhook := g.isWebhookHandler(route)

	// Get the request type - either from handler signature or webhook RequestSchema()
	requestType := route.RequestType
	if isWebhook && route.WebhookHandler != nil {
		if webhookRequestType := g.getWebhookRequestType(route.WebhookHandler); webhookRequestType != nil {
			requestType = *webhookRequestType
		}
	}

	// Process request sections using standard conventional logic
	if requestType.Kind() == reflect.Struct {
		g.processRequestSections(requestType, operation, components)
	} else if isWebhook && requestType.Kind() == reflect.Interface {
		// For webhook interface types, generate a generic request body
		g.addGenericWebhookRequestBody(operation)
	} else if isWebhook && requestType.Kind() != reflect.Invalid {
		// For webhook concrete non-struct request types (e.g., alias to provider event),
		// still add a request body schema to avoid missing request body in OpenAPI.
		operation.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]*MediaType{
				"application/json": {Schema: g.generateSchemaFromType(requestType, "", components)},
			},
		}
	}

	// Add webhook-specific metadata if this is a webhook
	if isWebhook {
		g.addWebhookMetadata(operation, route, components)
	}

	// Process response sections
	if route.ResponseType != nil {
		g.processResponseSections(route.ResponseType, operation, components, route)
	} else {
		// Error-only handlers generate 204 No Content
		operation.Responses["204"] = g.generateNoContentResponse()
	}

	// Add standard error responses to all operations
	g.addStandardErrorResponses(operation, components)

	return operation
}

// processRequestSections processes request sections for OpenAPI generation.
func (g *ConventionOpenAPIGenerator) processRequestSections(reqType reflect.Type, operation *Operation, components *Components) {
	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)

		switch field.Name {
		case SectionQuery:
			g.processQuerySection(field.Type, operation, components)
		case SectionPath:
			g.processPathSection(field.Type, operation, components)
		case SectionHeaders:
			g.processHeadersSection(field.Type, operation, components)
		case SectionBody:
			g.processBodySection(field.Type, reqType, operation, components)
		case SectionCookies:
			g.processCookiesSection(field.Type, operation, components)
		}
	}
}

// processQuerySection processes query parameters for OpenAPI.
func (g *ConventionOpenAPIGenerator) processQuerySection(sectionType reflect.Type, operation *Operation, components *Components) {
	if sectionType.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		gorkTag := field.Tag.Get("gork")
		validateTag := field.Tag.Get("validate")

		if gorkTag == "" {
			continue
		}

		tagInfo := parseGorkTag(gorkTag)

		var schema *Schema

		// Check if this field has a registered codec with schema information
		codecRegistry := gorkson.GetCodecRegistry()
		if codecSchema, exists := codecRegistry.Schema(field.Type); exists {
			schema = g.convertCodecSchemaToOpenAPI(codecSchema)
		} else {
			// Fall back to existing schema generation
			schema = g.generateSchemaFromType(field.Type, validateTag, components)
		}

		param := Parameter{
			Name:     tagInfo.Name,
			In:       "query",
			Required: strings.Contains(validateTag, "required"),
			Schema:   schema,
		}

		operation.Parameters = append(operation.Parameters, param)
	}
}

// processPathSection processes path parameters for OpenAPI.
func (g *ConventionOpenAPIGenerator) processPathSection(sectionType reflect.Type, operation *Operation, components *Components) {
	if sectionType.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		gorkTag := field.Tag.Get("gork")
		validateTag := field.Tag.Get("validate")

		if gorkTag == "" {
			continue
		}

		tagInfo := parseGorkTag(gorkTag)

		var schema *Schema

		// Check if this field has a registered codec with schema information
		codecRegistry := gorkson.GetCodecRegistry()
		if codecSchema, exists := codecRegistry.Schema(field.Type); exists {
			schema = g.convertCodecSchemaToOpenAPI(codecSchema)
		} else {
			// Fall back to existing schema generation
			schema = g.generateSchemaFromType(field.Type, validateTag, components)
		}

		param := Parameter{
			Name:     tagInfo.Name,
			In:       "path",
			Required: true, // Path parameters are always required
			Schema:   schema,
		}

		operation.Parameters = append(operation.Parameters, param)
	}
}

// convertCodecSchemaToOpenAPI converts internal OpenAPISchema to OpenAPI spec Schema.
func (g *ConventionOpenAPIGenerator) convertCodecSchemaToOpenAPI(codecSchema OpenAPISchema) *Schema {
	schema := &Schema{
		Type:        codecSchema.Type,
		Format:      codecSchema.Format,
		Pattern:     codecSchema.Pattern,
		MinLength:   codecSchema.MinLength,
		MaxLength:   codecSchema.MaxLength,
		Minimum:     codecSchema.Minimum,
		Maximum:     codecSchema.Maximum,
		Description: codecSchema.Description,
	}

	// Convert Enum from []interface{} to []string if present
	if codecSchema.Enum != nil {
		schema.Enum = make([]string, len(codecSchema.Enum))
		for i, val := range codecSchema.Enum {
			if str, ok := val.(string); ok {
				schema.Enum[i] = str
			} else {
				schema.Enum[i] = fmt.Sprintf("%v", val)
			}
		}
	}

	// Handle object properties if present
	if codecSchema.Properties != nil {
		schema.Properties = make(map[string]*Schema)
		for key, prop := range codecSchema.Properties {
			schema.Properties[key] = g.convertCodecSchemaToOpenAPI(*prop)
		}
	}

	// Handle array items if present
	if codecSchema.Items != nil {
		schema.Items = g.convertCodecSchemaToOpenAPI(*codecSchema.Items)
	}

	return schema
}

// processHeadersSection processes header parameters for OpenAPI.
func (g *ConventionOpenAPIGenerator) processHeadersSection(sectionType reflect.Type, operation *Operation, components *Components) {
	if sectionType.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		gorkTag := field.Tag.Get("gork")
		validateTag := field.Tag.Get("validate")

		if gorkTag == "" {
			continue
		}

		tagInfo := parseGorkTag(gorkTag)

		param := Parameter{
			Name:     tagInfo.Name,
			In:       "header",
			Required: strings.Contains(validateTag, "required"),
			Schema:   g.generateSchemaFromType(field.Type, validateTag, components),
		}

		operation.Parameters = append(operation.Parameters, param)
	}
}

// processCookiesSection processes cookie parameters for OpenAPI.
func (g *ConventionOpenAPIGenerator) processCookiesSection(sectionType reflect.Type, operation *Operation, components *Components) {
	if sectionType.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < sectionType.NumField(); i++ {
		field := sectionType.Field(i)
		gorkTag := field.Tag.Get("gork")
		validateTag := field.Tag.Get("validate")

		if gorkTag == "" {
			continue
		}

		tagInfo := parseGorkTag(gorkTag)

		param := Parameter{
			Name:     tagInfo.Name,
			In:       "cookie",
			Required: strings.Contains(validateTag, "required"),
			Schema:   g.generateSchemaFromType(field.Type, validateTag, components),
		}

		operation.Parameters = append(operation.Parameters, param)
	}
}

// processBodySection processes request body for OpenAPI.
func (g *ConventionOpenAPIGenerator) processBodySection(sectionType reflect.Type, reqType reflect.Type, operation *Operation, components *Components) {
	// Previously we returned early for non-struct Body, which prevented webhook
	// request bodies like raw []byte or external types from being represented.
	// Instead, always generate a schema for the body type: for structs it will
	// be a component/ref; for non-structs generate inline schema.

	var schema *Schema
	if sectionType.Kind() == reflect.Struct {
		// Generate component reference for the body section
		schema = g.generateRequestBodyComponentSchema(sectionType, reqType, components)
	} else {
		// Non-struct body (e.g., []byte, string, external aliases) → inline schema
		schema = g.generateInlineRequestBodySchema(sectionType, components)
	}

	operation.RequestBody = &RequestBody{
		Required: true,
		Content: map[string]*MediaType{
			"application/json": {
				Schema: schema,
			},
		},
	}
}

// processResponseSections processes response sections for OpenAPI using Convention Over Configuration.
func (g *ConventionOpenAPIGenerator) processResponseSections(respType reflect.Type, operation *Operation, components *Components, route *RouteInfo) {
	// Handle nil response type (error-only handlers)
	if respType == nil {
		operation.Responses["204"] = g.generateNoContentResponse()
		return
	}

	if respType.Kind() == reflect.Ptr {
		respType = respType.Elem()
	}

	// Handle non-struct responses
	if respType.Kind() != reflect.Struct {
		operation.Responses["204"] = g.generateNoContentResponse()
		return
	}

	// Handle empty structs (no fields) - should generate 204 No Content
	if respType.NumField() == 0 {
		operation.Responses["204"] = g.generateNoContentResponse()
		return
	}

	// All non-nil response types MUST follow Convention Over Configuration
	if !g.usesConventionSections(respType) {
		panic(fmt.Sprintf("response type must use Convention Over Configuration sections (Body, Headers, Cookies)\nHandler: %s %s -> %s\nResponse type: %s\nFound fields: %s",
			route.Method, route.Path, route.HandlerName, respType.Name(), g.getFieldNames(respType)))
	}

	hasBody := g.hasBodyField(respType)

	// Create response object to collect headers/cookies
	response := &Response{
		Description: "Success",
		Headers:     map[string]*Header{},
	}

	var bodySchema *Schema

	// Process response sections for headers, cookies, and body
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)

		switch field.Name {
		case SchemaSuffixBody.String():
			// Generate body schema only if there's a Body field
			if hasBody {
				bodySchema = g.generateResponseComponentSchema(respType, components)
			}
		case SchemaSuffixHeaders.String():
			g.processResponseHeaders(field.Type, response, components)
		case SchemaSuffixCookies.String():
			// Cookies are typically not documented in OpenAPI responses
			// They are set via Set-Cookie header
		}
	}

	// If there's no Body field, return 204 No Content (but with headers processed)
	if !hasBody {
		// Use 204 No Content response but preserve any headers that were processed
		noContentResponse := g.generateNoContentResponse()
		// Copy processed headers to the 204 response
		if len(response.Headers) > 0 {
			noContentResponse.Headers = response.Headers
		}
		operation.Responses["204"] = noContentResponse
		return
	}

	// Add body content for 200 response
	if bodySchema != nil {
		response.Content = map[string]*MediaType{
			"application/json": {
				Schema: bodySchema,
			},
		}
	}

	operation.Responses["200"] = response
}

// generateResponseComponentSchema creates a component reference for a response type,
// extracting properties from its Body field to create a clean schema.
func (g *ConventionOpenAPIGenerator) generateResponseComponentSchema(respType reflect.Type, components *Components) *Schema {
	typeName := respType.Name()
	if typeName == "" {
		// For anonymous types, we can't create a component reference
		// Fall back to inline schema generation
		return g.generateInlineResponseSchema(respType, components)
	}

	// Check if we already have this component
	if _, exists := components.Schemas[typeName]; exists {
		return &Schema{
			Ref: "#/components/schemas/" + typeName,
		}
	}

	// Find the Body field first to check if we should bypass the response wrapper
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)
		if field.Name == SchemaSuffixBody.String() {
			bodyType := field.Type

			// If Body is a named struct type, reference it directly instead of creating a wrapper
			if bodyType.Kind() == reflect.Struct && bodyType.Name() != "" && !isUnionType(bodyType) {
				// Generate schema for the body type directly
				return g.generateSchemaFromType(bodyType, "", components)
			}

			// For anonymous structs or other types, proceed with original logic
			break
		}
	}

	// Create the component schema by extracting Body field properties (fallback for complex cases)
	componentSchema := &Schema{
		Type:        "object",
		Title:       typeName,
		Properties:  make(map[string]*Schema),
		Description: g.getTypeDescription(respType),
	}

	// Find the Body field and extract its properties
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)
		if field.Name == SchemaSuffixBody.String() {
			// Extract properties from Body field type
			g.extractBodyPropertiesToResponseSchema(field.Type, componentSchema, components)
			break
		}
	}

	// Store the component schema with a collision-safe name
	unique := uniqueSchemaNameForType(respType, components.Schemas)
	componentSchema.Title = unique
	components.Schemas[unique] = componentSchema

	// Return a reference to the component
	return &Schema{
		Ref: "#/components/schemas/" + unique,
	}
}

// generateInlineResponseSchema generates an inline schema for anonymous response types.
func (g *ConventionOpenAPIGenerator) generateInlineResponseSchema(respType reflect.Type, components *Components) *Schema {
	// Find the Body field and generate its schema
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)
		if field.Name == SchemaSuffixBody.String() {
			return g.generateSchemaFromType(field.Type, "", components)
		}
	}
	return nil
}

// extractBodyPropertiesToResponseSchema extracts properties from a Body field type
// and adds them directly to the response component schema.
func (g *ConventionOpenAPIGenerator) extractBodyPropertiesToResponseSchema(bodyType reflect.Type, responseSchema *Schema, components *Components) {
	// For union types, the response schema becomes the union directly (no Body wrapper)
	// because the handler factory serializes only the Body field content, not the whole response
	if isUnionType(bodyType) {
		unionSchema := g.generateSchemaFromType(bodyType, "", components)
		if unionSchema != nil {
			// Copy the union schema properties directly to the response schema
			responseSchema.OneOf = unionSchema.OneOf
			responseSchema.Discriminator = unionSchema.Discriminator
			// Clear the type since we're using oneOf
			responseSchema.Type = ""
		}
		return
	}

	if bodyType.Kind() == reflect.Struct && bodyType.Name() != "" {
		// Named struct type - extract properties directly from the struct
		g.extractStructPropertiesToSchema(bodyType, responseSchema, components)
	} else {
		// Anonymous struct or other type - use existing schema generation
		bodySchema := g.generateSchemaFromType(bodyType, "", components)
		if bodySchema != nil && bodySchema.Properties != nil {
			// Copy properties from Body schema to the response schema
			for propName, propSchema := range bodySchema.Properties {
				responseSchema.Properties[propName] = propSchema
			}
			// Copy required fields too
			if bodySchema.Required != nil {
				responseSchema.Required = bodySchema.Required
			}
		}
	}
}

// extractStructPropertiesToSchema extracts properties from a struct type directly.
func (g *ConventionOpenAPIGenerator) extractStructPropertiesToSchema(structType reflect.Type, schema *Schema, components *Components) {
	// Don't extract fields from union types - they should be handled by union schema generation
	if isUnionType(structType) {
		return
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded structs - flatten their properties into parent schema
		if field.Anonymous && field.Type.Kind() == reflect.Struct && field.Tag.Get("json") == "" {
			g.flattenEmbeddedStructProperties(field.Type, schema, components)
			continue
		}

		// Get field name using proper tag parsing (gork -> json -> field name)
		fieldName := getOpenAPIFieldName(field)
		if fieldName == "" {
			continue // Skip fields marked as "-"
		}

		// Generate schema for the field
		fieldSchema := g.generateSchemaFromType(field.Type, field.Tag.Get("validate"), components)
		if fieldSchema != nil {
			schema.Properties[fieldName] = fieldSchema
		}

		// Check if field is required
		validateTag := field.Tag.Get("validate")
		if strings.Contains(validateTag, "required") {
			schema.Required = append(schema.Required, fieldName)
		}
	}
}

// flattenEmbeddedStructProperties flattens properties from an embedded struct into the parent schema.
func (g *ConventionOpenAPIGenerator) flattenEmbeddedStructProperties(embeddedType reflect.Type, parentSchema *Schema, components *Components) {
	// Recursively extract properties from the embedded struct
	for i := 0; i < embeddedType.NumField(); i++ {
		field := embeddedType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle nested embedded structs recursively
		if field.Anonymous && field.Type.Kind() == reflect.Struct && field.Tag.Get("json") == "" {
			g.flattenEmbeddedStructProperties(field.Type, parentSchema, components)
			continue
		}

		// Get field name using proper tag parsing (gork -> json -> field name)
		fieldName := getOpenAPIFieldName(field)
		if fieldName == "" {
			continue // Skip fields marked as "-"
		}

		// Generate schema for the field and add it to parent
		fieldSchema := g.generateSchemaFromType(field.Type, field.Tag.Get("validate"), components)
		if fieldSchema != nil {
			parentSchema.Properties[fieldName] = fieldSchema
		}

		// Check if field is required and add to parent's required list
		validateTag := field.Tag.Get("validate")
		if strings.Contains(validateTag, "required") {
			parentSchema.Required = append(parentSchema.Required, fieldName)
		}
	}
}

// generateRequestBodyComponentSchema creates a component reference for a request body section.
func (g *ConventionOpenAPIGenerator) generateRequestBodyComponentSchema(bodyType reflect.Type, reqType reflect.Type, components *Components) *Schema {
	// For request bodies, we want to create a component schema for the body content
	// We'll use the request context to generate a meaningful component name
	componentName := g.generateRequestBodyComponentName(bodyType, reqType)

	if componentName == "" {
		// For anonymous types with no meaningful name, fall back to inline schema generation
		return g.generateInlineRequestBodySchema(bodyType, components)
	}

	// Check if we already have this component
	if _, exists := components.Schemas[componentName]; exists {
		return &Schema{
			Ref: "#/components/schemas/" + componentName,
		}
	}

	// Create the component schema directly from the body type
	componentSchema := &Schema{
		Type:        "object",
		Title:       componentName,
		Properties:  make(map[string]*Schema),
		Description: "",
	}

	// Extract properties from the body struct
	g.extractStructPropertiesToSchema(bodyType, componentSchema, components)

	// Store the component schema
	components.Schemas[componentName] = componentSchema

	// Return a reference to the component
	return &Schema{
		Ref: "#/components/schemas/" + componentName,
	}
}

// generateRequestBodyComponentName generates a component name for a request body.
func (g *ConventionOpenAPIGenerator) generateRequestBodyComponentName(bodyType reflect.Type, reqType reflect.Type) string {
	// For request body schemas, we'll generate names based on the request context
	// to create descriptive component names like "UpdateUserPreferencesBody"

	if bodyType.Name() != "" {
		// If the body type has a name, use it with "Body" suffix
		// For union types, create more concise names
		if isUnionType(bodyType) {
			return g.generateConciseUnionName(bodyType) + SchemaSuffixBody.String()
		}
		// Apply sanitization to handle other complex names
		return sanitizeSchemaName(bodyType.Name()) + SchemaSuffixBody.String()
	}

	// For anonymous structs, use the parent request name for context
	if reqType != nil && reqType.Name() != "" {
		requestName := reqType.Name()
		// Remove "Request" suffix if present to avoid "RequestBody"
		requestName = strings.TrimSuffix(requestName, "Request")
		return sanitizeSchemaName(requestName) + SchemaSuffixBody.String()
	}

	// Fallback: for anonymous structs with no request context, create name based on field names
	var fieldNames []string
	for i := 0; i < bodyType.NumField() && i < 3; i++ { // Use up to 3 field names
		field := bodyType.Field(i)
		if field.IsExported() {
			fieldNames = append(fieldNames, field.Name)
		}
	}

	if len(fieldNames) > 0 {
		return strings.Join(fieldNames, "") + "Request"
	}

	// Return empty string to indicate that no meaningful component name can be generated
	// This will trigger inline schema generation
	return ""
}

// generateInlineRequestBodySchema generates an inline schema for request body types
// when no meaningful component name can be generated.
func (g *ConventionOpenAPIGenerator) generateInlineRequestBodySchema(bodyType reflect.Type, components *Components) *Schema {
	// Generate schema directly from the body type without creating a component
	return g.generateSchemaFromType(bodyType, "", components)
}

// getTypeDescription extracts documentation from a type's comment using the generator's doc extractor.
func (g *ConventionOpenAPIGenerator) getTypeDescription(t reflect.Type) string {
	if g.extractor == nil {
		return ""
	}

	typeName := t.Name()
	if typeName == "" {
		return ""
	}

	doc := g.extractor.ExtractTypeDoc(typeName)
	return doc.Description
}

// processResponseHeaders processes response headers for OpenAPI.
func (g *ConventionOpenAPIGenerator) processResponseHeaders(headersType reflect.Type, response *Response, components *Components) {
	if headersType.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < headersType.NumField(); i++ {
		field := headersType.Field(i)
		gorkTag := field.Tag.Get("gork")

		if gorkTag == "" {
			continue
		}

		tagInfo := parseGorkTag(gorkTag)

		header := &Header{
			Description: "Response header",
			Schema:      g.generateSchemaFromType(field.Type, "", components),
		}

		response.Headers[tagInfo.Name] = header
	}
}

// generateSchemaFromType generates OpenAPI schema from Go type with union support.
func (g *ConventionOpenAPIGenerator) generateSchemaFromType(fieldType reflect.Type, validateTag string, components *Components) *Schema {
	// Handle nil types gracefully
	if fieldType == nil {
		return nil
	}

	// Check if this is a union type
	if isUnionType(fieldType) {
		return g.generateUnionSchema(fieldType, components)
	}

	// Handle other types using existing logic
	schema := reflectTypeToSchema(fieldType, components.Schemas)
	if schema != nil && validateTag != "" {
		// Create a dummy struct field for validation constraints
		sf := reflect.StructField{
			Name: "Field",
			Type: fieldType,
			Tag:  reflect.StructTag(`validate:"` + validateTag + `"`),
		}
		// Create a dummy parent schema for validation constraints
		parentSchema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
			Required:   []string{},
		}
		applyValidationConstraints(schema, validateTag, fieldType, parentSchema, sf)
	}
	return schema
}

// usesConventionSections checks if a struct type uses Convention Over Configuration sections.
func (g *ConventionOpenAPIGenerator) usesConventionSections(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}

	// Check if any field uses standard section names
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if AllowedSections[field.Name] {
			return true
		}
	}

	return false
}

// generateUnionSchema generates OpenAPI oneOf schema for union types.
func (g *ConventionOpenAPIGenerator) generateUnionSchema(unionType reflect.Type, components *Components) *Schema {
	// Extract union member types from the union struct
	unionTypes := g.extractUnionMemberTypes(unionType)
	if len(unionTypes) == 0 {
		// Fallback for unknown union structure
		return &Schema{Type: "object", Description: "Unknown union type"}
	}

	oneOfSchemas, discriminatorMapping := g.generateUnionMemberSchemas(unionTypes, components)

	schema := &Schema{
		OneOf: oneOfSchemas,
	}

	// Add discriminator if we found discriminator values
	if len(discriminatorMapping) > 0 {
		schema.Discriminator = &Discriminator{
			PropertyName: "type",
			Mapping:      discriminatorMapping,
		}
	}

	return schema
}

// generateUnionMemberSchemas generates schemas for union member types and discriminator mapping.
func (g *ConventionOpenAPIGenerator) generateUnionMemberSchemas(unionTypes []reflect.Type, components *Components) ([]*Schema, map[string]string) {
	oneOfSchemas := make([]*Schema, 0, len(unionTypes))
	discriminatorMapping := make(map[string]string)

	for _, memberType := range unionTypes {
		memberSchema := g.generateSchemaFromType(memberType, "", components)
		if memberSchema == nil {
			continue
		}

		oneOfSchemas = append(oneOfSchemas, memberSchema)
		g.addDiscriminatorMapping(memberType, discriminatorMapping)
	}

	return oneOfSchemas, discriminatorMapping
}

// addDiscriminatorMapping adds discriminator mapping for a member type if applicable.
func (g *ConventionOpenAPIGenerator) addDiscriminatorMapping(memberType reflect.Type, discriminatorMapping map[string]string) {
	if memberType.Kind() != reflect.Struct {
		return
	}

	discriminatorValue := g.extractDiscriminatorValue(memberType)
	if discriminatorValue == "" {
		return
	}

	typeName := sanitizeSchemaName(memberType.Name())
	if typeName != "" {
		discriminatorMapping[discriminatorValue] = "#/components/schemas/" + typeName
	}
}

// extractUnionMemberTypes extracts the member types from a union type.
func (g *ConventionOpenAPIGenerator) extractUnionMemberTypes(unionType reflect.Type) []reflect.Type {
	var memberTypes []reflect.Type

	if unionType.Kind() != reflect.Struct {
		return memberTypes
	}

	// Union types have all fields as union member pointers by definition
	for i := 0; i < unionType.NumField(); i++ {
		field := unionType.Field(i)
		// Only process pointer fields (union members)
		if field.Type.Kind() == reflect.Ptr {
			// Get the element type (the actual union member type)
			memberTypes = append(memberTypes, field.Type.Elem())
		}
	}

	return memberTypes
}

// extractDiscriminatorValue extracts discriminator value from struct tags.
func (g *ConventionOpenAPIGenerator) extractDiscriminatorValue(structType reflect.Type) string {
	if structType.Kind() != reflect.Struct {
		return ""
	}

	// Look for a field with discriminator tag
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if discriminatorValue := g.parseDiscriminatorFromGorkTag(field.Tag.Get("gork")); discriminatorValue != "" {
			return discriminatorValue
		}
	}

	return ""
}

// parseDiscriminatorFromGorkTag parses discriminator value from gork tag.
func (g *ConventionOpenAPIGenerator) parseDiscriminatorFromGorkTag(gorkTag string) string {
	if gorkTag == "" || !strings.Contains(gorkTag, "discriminator=") {
		return ""
	}

	parts := strings.Split(gorkTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "discriminator=") {
			return strings.TrimPrefix(part, "discriminator=")
		}
	}

	return ""
}

// getFieldNames returns a formatted string listing all field names in a struct type.
func (g *ConventionOpenAPIGenerator) getFieldNames(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return fmt.Sprintf("not a struct (kind: %s)", t.Kind())
	}

	var fields []string
	for i := 0; i < t.NumField(); i++ {
		fields = append(fields, t.Field(i).Name)
	}

	if len(fields) == 0 {
		return "no fields"
	}

	return strings.Join(fields, ", ")
}

// hasBodyField checks if a struct type has a Body field.
func (g *ConventionOpenAPIGenerator) hasBodyField(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Name == SchemaSuffixBody.String() {
			return true
		}
	}
	return false
}

// generateNoContentResponse generates a 204 No Content response for OpenAPI.
func (g *ConventionOpenAPIGenerator) generateNoContentResponse() *Response {
	return &Response{
		Description: "No Content",
	}
}

// addStandardErrorResponses adds the standard error responses (400, 422, 500) to all operations.
func (g *ConventionOpenAPIGenerator) addStandardErrorResponses(operation *Operation, components *Components) {
	// Ensure standard error response components exist
	ensureStdResponses(components)

	// Ensure error response schemas exist in components
	g.ensureErrorSchemas(components)

	// Add standard error responses to the operation
	if operation.Responses == nil {
		operation.Responses = map[string]*Response{}
	}

	// 400 Bad Request - Validation failed
	if _, exists := operation.Responses["400"]; !exists {
		operation.Responses["400"] = &Response{Ref: "#/components/responses/BadRequest"}
	}

	// 422 Unprocessable Entity - Request body could not be parsed
	if _, exists := operation.Responses["422"]; !exists {
		operation.Responses["422"] = &Response{Ref: "#/components/responses/UnprocessableEntity"}
	}

	// 500 Internal Server Error
	if _, exists := operation.Responses["500"]; !exists {
		operation.Responses["500"] = &Response{Ref: "#/components/responses/InternalServerError"}
	}
}

// ensureErrorSchemas ensures that ErrorResponse and ValidationErrorResponse schemas exist in components.
func (g *ConventionOpenAPIGenerator) ensureErrorSchemas(components *Components) {
	if components.Schemas == nil {
		components.Schemas = map[string]*Schema{}
	}

	// Add ErrorResponse schema if it doesn't exist
	if _, exists := components.Schemas["ErrorResponse"]; !exists {
		components.Schemas["ErrorResponse"] = &Schema{
			Type:        "object",
			Title:       "ErrorResponse",
			Description: "Generic error response structure",
			Properties: map[string]*Schema{
				"error": {
					Type:        "string",
					Description: "Error message",
				},
				"details": {
					Type:        "object",
					Description: "Additional error details",
				},
			},
			Required: []string{"error"},
		}
	}

	// Add ValidationErrorResponse schema if it doesn't exist
	if _, exists := components.Schemas["ValidationErrorResponse"]; !exists {
		components.Schemas["ValidationErrorResponse"] = &Schema{
			Type:        "object",
			Title:       "ValidationErrorResponse",
			Description: "Validation error response with field-level details",
			Properties: map[string]*Schema{
				"error": {
					Type:        "string",
					Description: "Error message",
				},
				"details": {
					Type:        "object",
					Description: "Field-level validation errors (maps field names to arrays of error messages)",
				},
			},
			Required: []string{"error"},
		}
	}
}

// generateConciseUnionName creates shorter, more readable names for union types.
func (g *ConventionOpenAPIGenerator) generateConciseUnionName(bodyType reflect.Type) string {
	typeName := bodyType.Name()

	// Extract union type info: "Union2[A,B]" -> base="Union2", members=["A", "B"]
	open := strings.Index(typeName, "[")
	if open == -1 || !strings.HasSuffix(typeName, "]") {
		// Not a generic union type, fall back to sanitization
		return sanitizeSchemaName(typeName)
	}

	base := typeName[:open]
	args := typeName[open+1 : len(typeName)-1]
	parts := strings.Split(args, ",")

	// Clean up member names and extract short identifiers
	cleanMembers := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Strip package path: "github.com/gork-labs/gork/examples/handlers.BankPaymentMethod" -> "BankPaymentMethod"
		if idx := strings.LastIndex(p, "."); idx != -1 {
			p = p[idx+1:]
		}
		// Extract meaningful part of the name (remove common suffixes/prefixes)
		p = g.extractMeaningfulName(p)
		cleanMembers = append(cleanMembers, p)
	}

	// Create a concise name based on the pattern
	return g.createUnionNameByMemberCount(base, cleanMembers)
}

// createUnionNameByMemberCount creates a union name based on the number of cleaned members.
func (g *ConventionOpenAPIGenerator) createUnionNameByMemberCount(base string, cleanMembers []string) string {
	switch len(cleanMembers) {
	case 2:
		// For two members, create an "Or" pattern or short combination
		return g.createBinaryUnionName(cleanMembers)
	case 3, 4:
		// For more members, use a numbered approach
		return base + "Options"
	default:
		// Fallback to the base union type
		return base + "Type"
	}
}

// extractMeaningfulName extracts the most meaningful part of a type name.
func (g *ConventionOpenAPIGenerator) extractMeaningfulName(name string) string {
	// Remove common suffixes to get the core concept
	suffixes := []string{"PaymentMethod", "Payment", "Method", "Auth", "Request", "Response", "Type"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) && len(name) > len(suffix) {
			core := strings.TrimSuffix(name, suffix)
			if len(core) > 2 { // Keep meaningful cores only
				return core
			}
		}
	}

	// If no suffix matches, use the first part or whole name if short
	if len(name) <= 8 {
		return name
	}

	// For longer names, try to extract the first meaningful word
	// CreditCardPaymentMethod -> CreditCard
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' && i > 4 {
			return name[:i]
		}
	}

	return name
}

// createBinaryUnionName creates names for 2-member unions.
func (g *ConventionOpenAPIGenerator) createBinaryUnionName(members []string) string {
	if len(members) != 2 {
		return "UnionType"
	}

	// Special patterns for common combinations
	a, b := members[0], members[1]

	// If both names are short, combine them
	if len(a) <= 6 && len(b) <= 6 {
		return a + "Or" + b
	}

	// If one is much shorter, use the pattern "ShortOrLong"
	if len(a) <= 4 {
		return a + "Or" + b[:min(6, len(b))]
	}
	if len(b) <= 4 {
		return a[:min(6, len(a))] + "Or" + b
	}

	// For longer names, use abbreviated form
	return a[:min(4, len(a))] + "Or" + b[:min(4, len(b))]
}

// isWebhookHandler determines webhook handlers by presence of webhook-related metadata.
// Routes created via api.WebhookHandlerFunc populate RouteInfo.WebhookHandler.
// Routes can also be detected by having WebhookProviderInfo or WebhookHandledEvents.
func (g *ConventionOpenAPIGenerator) isWebhookHandler(route *RouteInfo) bool {
	if route == nil {
		return false
	}
	hasHandler := route.WebhookHandler != nil
	hasProviderInfo := route.WebhookProviderInfo != nil
	hasEvents := len(route.WebhookHandledEvents) > 0

	result := hasHandler || hasProviderInfo || hasEvents

	return result
}

// addWebhookExtensions adds webhook-specific extensions to the operation.
func (g *ConventionOpenAPIGenerator) addWebhookExtensions(operation *Operation, route *RouteInfo) {
	// Set summary and description for webhook operations
	operation.Summary = fmt.Sprintf("Webhook endpoint for %s", route.HandlerName)
	operation.Description = "Webhook endpoint that receives events from external services"

	// Add webhook-specific extensions
	if operation.Extensions == nil {
		operation.Extensions = make(map[string]interface{})
	}

	// Attach provider metadata
	if p := g.getWebhookProviderInfo(route); p != nil {
		provider := map[string]string{"name": p.Name, "website": p.Website, "docs": p.DocsURL}
		operation.Extensions["x-webhook-provider"] = provider
		operation.XWebhookProvider = provider
	}

	// Add webhook event metadata
	eventEntries := g.buildWebhookEventEntries(route, g.spec.Components)
	if len(eventEntries) > 0 {
		operation.Extensions["x-webhook-events"] = eventEntries
		operation.XWebhookEvents = eventEntries
	}
}

// getWebhookProvider determines the webhook provider from the request type.
// Note: Provider detection is intentionally omitted. Configuration is user-driven.
func (g *ConventionOpenAPIGenerator) getWebhookProviderInfo(route *RouteInfo) *WebhookProviderInfo {
	if route.WebhookProviderInfo != nil {
		return route.WebhookProviderInfo
	}
	if route.WebhookHandler == nil {
		return nil
	}
	hv := reflect.ValueOf(route.WebhookHandler)
	m := hv.MethodByName("ProviderInfo")
	if !m.IsValid() {
		return nil
	}
	res := m.Call(nil)
	if len(res) == 0 {
		return nil
	}
	if pi, ok := res[0].Interface().(WebhookProviderInfo); ok {
		return &pi
	}
	return nil
}

// getWebhookEventTypes extracts supported event types from the route using reflection.
func (g *ConventionOpenAPIGenerator) getWebhookEventTypes(route *RouteInfo) []string {
	if route.WebhookHandler == nil {
		return nil
	}

	return g.extractEventTypesFromHandler(route.WebhookHandler)
}

// extractEventTypesFromHandler extracts event types from a webhook handler using reflection.
func (g *ConventionOpenAPIGenerator) extractEventTypesFromHandler(handler interface{}) []string {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	method, exists := handlerType.MethodByName("GetValidEventTypes")
	if !exists {
		return nil
	}

	if !g.isValidEventTypesMethod(method.Type) {
		return nil
	}

	return g.callEventTypesMethod(handlerValue)
}

// isValidEventTypesMethod checks if the method has the correct signature.
func (g *ConventionOpenAPIGenerator) isValidEventTypesMethod(methodType reflect.Type) bool {
	return methodType.NumIn() == 1 && methodType.NumOut() == 1
}

// callEventTypesMethod calls the GetValidEventTypes method and returns the result.
func (g *ConventionOpenAPIGenerator) callEventTypesMethod(handlerValue reflect.Value) []string {
	results := handlerValue.MethodByName("GetValidEventTypes").Call(nil)
	if len(results) == 0 {
		return nil
	}

	eventTypes, ok := results[0].Interface().([]string)
	if !ok {
		return nil
	}

	return eventTypes
}

// getWebhookRequestType gets the request type from the webhook handler's RequestSchema() method.
func (g *ConventionOpenAPIGenerator) getWebhookRequestType(handler interface{}) *reflect.Type {
	if handler == nil {
		return nil
	}

	// Use reflection to call the RequestSchema() method
	handlerValue := reflect.ValueOf(handler)
	requestSchemaMethod := handlerValue.MethodByName("RequestSchema")

	if !requestSchemaMethod.IsValid() {
		return nil
	}

	// Call the RequestSchema() method
	results := requestSchemaMethod.Call([]reflect.Value{})
	if len(results) != 1 {
		return nil
	}

	schemaType, ok := results[0].Interface().(reflect.Type)
	if !ok {
		return nil
	}

	return &schemaType
}

// getWebhookResponseType gets the response type from the webhook handler's ResponseSchema() method.
func (g *ConventionOpenAPIGenerator) getWebhookResponseType(handler interface{}) *reflect.Type {
	if handler == nil {
		return nil
	}

	// Use reflection to call the ResponseSchema() method
	handlerValue := reflect.ValueOf(handler)
	responseSchemaMethod := handlerValue.MethodByName("ResponseSchema")

	if !responseSchemaMethod.IsValid() {
		return nil
	}

	// Call the ResponseSchema() method
	results := responseSchemaMethod.Call([]reflect.Value{})
	if len(results) != 1 {
		return nil
	}

	schemaType, ok := results[0].Interface().(reflect.Type)
	if !ok {
		return nil
	}

	return &schemaType
}

// extractBodySchemaFromConventionalResponse extracts the Body field schema from a conventional response struct.
func (g *ConventionOpenAPIGenerator) extractBodySchemaFromConventionalResponse(respType reflect.Type, components *Components) *Schema {
	if respType.Kind() == reflect.Ptr {
		respType = respType.Elem()
	}

	if respType.Kind() != reflect.Struct {
		return nil
	}

	// Look for the Body field
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)
		if field.Name == "Body" {
			// Found Body field, generate schema using struct field names (not tags)
			if field.Type.Kind() == reflect.Struct {
				return g.generateStructSchemaUsingFieldNames(field.Type, components)
			}
			return g.generateSchemaFromType(field.Type, "", components)
		}
		// Also support conventional field tagged as body via gork/json tags
		if tag := field.Tag.Get("gork"); tag != "" {
			name := parseGorkTag(tag).Name
			if name == "body" {
				return g.generateSchemaFromType(field.Type, "", components)
			}
		}
		if jsonTag := field.Tag.Get("json"); strings.HasPrefix(jsonTag, "body") {
			return g.generateSchemaFromType(field.Type, "", components)
		}
	}

	return nil
}

// generateStructSchemaUsingFieldNames builds an object schema using struct field identifiers
// as property names, ignoring gork/json tags. This is used for webhook reflected responses
// to satisfy expected naming in tests and documentation.
func (g *ConventionOpenAPIGenerator) generateStructSchemaUsingFieldNames(structType reflect.Type, components *Components) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: map[string]*Schema{},
	}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		// Use the Go identifier directly
		name := field.Name
		fieldSchema := g.generateSchemaFromType(field.Type, field.Tag.Get("validate"), components)
		if fieldSchema != nil {
			schema.Properties[name] = fieldSchema
		}
	}
	return schema
}

// addWebhookMetadata adds webhook-specific metadata to the operation.
func (g *ConventionOpenAPIGenerator) addWebhookMetadata(operation *Operation, route *RouteInfo, components *Components) {
	// Add webhook-specific responses (handles fallback case internally)
	g.addWebhookResponses(operation, components, route)

	// Add webhook provider extensions (events, provider info, etc.)
	g.addWebhookExtensions(operation, route)
}

// addGenericWebhookRequestBody adds a generic request body for webhook interface types.
func (g *ConventionOpenAPIGenerator) addGenericWebhookRequestBody(operation *Operation) {
	operation.RequestBody = &RequestBody{
		Required: true,
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Type:        "object",
					Description: "Webhook event payload",
				},
			},
		},
		Description: "Webhook event payload",
	}
}

// buildWebhookResponseSchema builds a schema for webhook response bodies.
// For webhooks, this extracts the Body section from the provider's ResponseSchema.
func (g *ConventionOpenAPIGenerator) buildWebhookResponseSchema(respType reflect.Type, components *Components, isSuccess bool, route *RouteInfo) *Schema {
	// Try to get provider-specific response schema from the webhook handler
	if route != nil && route.WebhookHandler != nil {
		if responseType := g.getWebhookResponseType(route.WebhookHandler); responseType != nil {
			// Extract Body section from the ResponseSchema
			if bodySchema := g.extractBodySchemaFromConventionalResponse(*responseType, components); bodySchema != nil {
				return bodySchema
			}
			// Fallback: use full conventional response struct to produce schema
			if (*responseType).Kind() == reflect.Struct {
				return g.generateSchemaFromType(*responseType, "", components)
			}
		}
	}

	// Fallback to existing logic for backward compatibility
	if respType == nil {
		return &Schema{Type: "object"}
	}

	// Try to extract fields from the Body wrapper if it exists
	if respType.Kind() == reflect.Struct {
		for i := 0; i < respType.NumField(); i++ {
			field := respType.Field(i)
			if field.Name == "Body" {
				// Found Body field, extract its properties directly
				return g.generateSchemaFromType(field.Type, "", components)
			}
		}
	}

	// If no Body field found, but it's a struct, use the struct directly (custom response types)
	if respType.Kind() == reflect.Struct {
		return g.generateSchemaFromType(respType, "", components)
	}

	// Fallback to generic webhook response for non-struct types
	if isSuccess {
		return &Schema{
			Type:        "object",
			Description: "Webhook processed successfully",
			Properties: map[string]*Schema{
				"received": {
					Type:        "boolean",
					Description: "Indicates if the webhook was received and processed",
				},
			},
		}
	} else {
		return &Schema{
			Type:        "object",
			Description: "Webhook processing error",
			Properties: map[string]*Schema{
				"received": {
					Type:        "boolean",
					Description: "Indicates if the webhook was received",
				},
				"error": {
					Type:        "string",
					Description: "Error message describing what went wrong",
				},
			},
		}
	}
}

// getWebhookProviderResponseBodySchema gets the response body schema from the provider's ResponseSchema.
func (g *ConventionOpenAPIGenerator) getWebhookProviderResponseBodySchema(handler interface{}, components *Components) *Schema {
	if handler == nil {
		return nil
	}

	// Use reflection to call the ResponseSchema() method on the webhook handler
	handlerValue := reflect.ValueOf(handler)
	responseSchemaMethod := handlerValue.MethodByName("ResponseSchema")

	if !responseSchemaMethod.IsValid() {
		return nil
	}

	// Call the ResponseSchema() method
	results := responseSchemaMethod.Call([]reflect.Value{})
	if len(results) != 1 {
		return nil
	}

	schemaType, ok := results[0].Interface().(reflect.Type)
	if !ok || schemaType == nil {
		return nil
	}

	// Extract the Body field from the response schema
	return g.extractBodySchemaFromConventionalResponse(schemaType, components)
}

// providerFromPkgPath extracts the provider name from a package path ending with "/webhooks/<provider>".
func (g *ConventionOpenAPIGenerator) providerFromPkgPath(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	// Split path and get last segment
	parts := strings.Split(pkgPath, "/")
	last := parts[len(parts)-1]
	// If the last segment is "webhooks", try the previous one
	if last == "webhooks" && len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return last
}

// toPascalCase converts a string like "stripe" or "stripe_webhook" to "StripeWebhook".
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	// Split by non-alphanumeric boundaries
	sep := func(r rune) bool { return r == '_' || r == '-' || r == ' ' || r == '.' }
	parts := strings.FieldsFunc(s, sep)
	out := ""
	for _, p := range parts {
		out += strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return out
}

// addWebhookResponses adds webhook-specific response schemas using reflection on the actual handler instance.
// This method uses reflection to call SuccessResponse() and ErrorResponse() methods on the webhook handler
// to generate accurate OpenAPI response schemas instead of hardcoded placeholders.
func (g *ConventionOpenAPIGenerator) addWebhookResponses(operation *Operation, components *Components, route *RouteInfo) {
	// Extract the original webhook handler instance from the route
	webhookHandler := route.WebhookHandler
	if webhookHandler == nil {
		// Fallback to basic responses if no webhook handler is available
		g.addFallbackWebhookResponses(operation, components)
		return
	}

	// Use webhook's ResponseSchema() method to generate response schemas
	successSchema := g.buildWebhookResponseSchema(nil, components, true, route)
	operation.Responses["200"] = &Response{
		Description: "Webhook processed successfully",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: successSchema,
			},
		},
	}

	// Generate error response schema
	errorSchema := g.buildWebhookResponseSchema(nil, components, false, route)
	operation.Responses["400"] = &Response{
		Description: "Invalid webhook payload or signature",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: errorSchema,
			},
		},
	}
}

// providerSpecificWebhookTypeRef ensures provider-prefixed component names for generic webhook types.
func (g *ConventionOpenAPIGenerator) providerSpecificWebhookTypeRef(t reflect.Type, schema *Schema, components *Components, suffix string) *Schema {
	pkgPath := t.PkgPath()
	provider := g.providerFromPkgPath(pkgPath)
	if provider == "" {
		return schema
	}
	compName := toPascalCase(provider) + suffix
	if components.Schemas == nil {
		components.Schemas = map[string]*Schema{}
	}
	var toStore Schema
	if schema != nil {
		if schema.Ref != "" {
			refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
			if resolved, ok := components.Schemas[refName]; ok && resolved != nil {
				toStore = *resolved
			} else {
				toStore = *schema
			}
		} else {
			toStore = *schema
		}
		toStore.Title = compName
		components.Schemas[compName] = &toStore
		// Remove generic component counterpart if present
		delete(components.Schemas, suffix)
		return &Schema{Ref: "#/components/schemas/" + compName}
	}
	return schema
}

// buildWebhookEventsMetadata constructs an array of objects describing each registered event handler.
// For each event we include:
// - event: string
// - description: pulled from function doc (if available)
// - operationId: function name
// - userPayloadSchema: JSON schema for the user metadata parameter (if any).
func (g *ConventionOpenAPIGenerator) buildWebhookEventsMetadata(route *RouteInfo, components *Components) []map[string]interface{} {
	metas := route.WebhookHandlersMeta
	if len(metas) == 0 {
		return nil
	}

	out := make([]map[string]interface{}, 0, len(metas))
	for _, m := range metas {
		entry := g.buildSingleEventMetadata(m, components)
		out = append(out, entry)
	}
	return out
}

// buildSingleEventMetadata builds metadata for a single webhook event.
func (g *ConventionOpenAPIGenerator) buildSingleEventMetadata(meta RegisteredEventHandler, components *Components) map[string]interface{} {
	entry := map[string]interface{}{
		"event": meta.EventType,
		// operationId: name of the registered handler function (closest stable identifier)
		"operationId": meta.HandlerName,
	}

	g.addDescriptionToEntry(entry, meta.HandlerName)
	g.addUserMetadataSchemaToEntry(entry, meta.UserMetadataType, components)

	return entry
}

// addDescriptionToEntry adds description from function docs if available.
func (g *ConventionOpenAPIGenerator) addDescriptionToEntry(entry map[string]interface{}, handlerName string) {
	if g.extractor == nil || handlerName == "" {
		return
	}

	fd := g.extractor.ExtractFunctionDoc(handlerName)
	if fd.Description != "" {
		entry["description"] = fd.Description
	}
}

// addUserMetadataSchemaToEntry adds user metadata schema if present.
func (g *ConventionOpenAPIGenerator) addUserMetadataSchemaToEntry(entry map[string]interface{}, userMetadataType reflect.Type, components *Components) {
	if userMetadataType == nil {
		return
	}

	userT := userMetadataType
	if userT.Kind() == reflect.Ptr {
		userT = userT.Elem()
	}

	if userT.Kind() != reflect.Invalid {
		schema := g.generateSchemaFromType(userT, "", components)
		entry["userPayloadSchema"] = schema
	}
}

// resolveConcreteType resolves the concrete type from an interface{} return type.
func (g *ConventionOpenAPIGenerator) resolveConcreteType(method reflect.Value, methodType reflect.Type, returnType reflect.Type, methodName string) reflect.Type {
	if returnType.Kind() != reflect.Interface || returnType.String() != "interface {}" {
		return returnType
	}

	switch methodName {
	case "SuccessResponse":
		return g.getSuccessResponseType(method, methodType)
	case "ErrorResponse":
		return g.getErrorResponseType(method, methodType)
	default:
		return returnType
	}
}

// getSuccessResponseType gets the concrete type for SuccessResponse method.
func (g *ConventionOpenAPIGenerator) getSuccessResponseType(method reflect.Value, methodType reflect.Type) reflect.Type {
	if methodType.NumIn() != 0 {
		// Return the static return type even with wrong signature
		return methodType.Out(0)
	}

	results := method.Call(nil)
	return g.extractTypeFromResults(results)
}

// getErrorResponseType gets the concrete type for ErrorResponse method.
func (g *ConventionOpenAPIGenerator) getErrorResponseType(method reflect.Value, methodType reflect.Type) reflect.Type {
	if methodType.NumIn() != 1 {
		// Return the static return type even with wrong signature
		return methodType.Out(0)
	}

	errorArg := reflect.ValueOf(fmt.Errorf("sample error"))
	results := method.Call([]reflect.Value{errorArg})
	return g.extractTypeFromResults(results)
}

// extractTypeFromResults extracts the concrete type from method call results.
func (g *ConventionOpenAPIGenerator) extractTypeFromResults(results []reflect.Value) reflect.Type {
	if len(results) == 0 || !results[0].IsValid() {
		return nil
	}

	result := results[0]
	if result.Kind() == reflect.Interface && !result.IsNil() {
		return result.Elem().Type()
	}

	return result.Type()
}

// buildWebhookEventEntries builds webhook event entries based on available metadata.
func (g *ConventionOpenAPIGenerator) buildWebhookEventEntries(route *RouteInfo, components *Components) []map[string]interface{} {
	switch {
	case len(route.WebhookHandlersMeta) > 0:
		return g.buildWebhookEventsMetadata(route, components)
	case len(route.WebhookHandledEvents) > 0:
		return g.buildEventEntriesFromHandledEvents(route.WebhookHandledEvents)
	default:
		return g.buildEventEntriesFromProvider(route)
	}
}

// buildEventEntriesFromHandledEvents builds event entries from handled events list.
func (g *ConventionOpenAPIGenerator) buildEventEntriesFromHandledEvents(handledEvents []string) []map[string]interface{} {
	eventEntries := make([]map[string]interface{}, 0, len(handledEvents))
	for _, evt := range handledEvents {
		eventEntries = append(eventEntries, map[string]interface{}{"event": evt})
	}
	return eventEntries
}

// buildEventEntriesFromProvider builds event entries from provider-advertised list.
func (g *ConventionOpenAPIGenerator) buildEventEntriesFromProvider(route *RouteInfo) []map[string]interface{} {
	eventTypes := g.getWebhookEventTypes(route)
	if len(eventTypes) == 0 {
		return nil
	}

	eventEntries := make([]map[string]interface{}, 0, len(eventTypes))
	for _, evt := range eventTypes {
		eventEntries = append(eventEntries, map[string]interface{}{"event": evt})
	}
	return eventEntries
}

// addFallbackWebhookResponses adds basic webhook responses when reflection fails.
func (g *ConventionOpenAPIGenerator) addFallbackWebhookResponses(operation *Operation, _ *Components) {
	operation.Responses["200"] = g.createFallbackSuccessResponse()
	operation.Responses["400"] = g.createFallbackErrorResponse()
}

// createFallbackSuccessResponse creates a basic success response for webhooks.
func (g *ConventionOpenAPIGenerator) createFallbackSuccessResponse() *Response {
	return &Response{
		Description: "Webhook processed successfully",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"received": {
							Type:        "boolean",
							Description: "Whether the webhook was received successfully",
						},
					},
					Required: []string{"received"},
				},
			},
		},
	}
}

// createFallbackErrorResponse creates a basic error response for webhooks.
func (g *ConventionOpenAPIGenerator) createFallbackErrorResponse() *Response {
	return &Response{
		Description: "Invalid webhook payload or signature",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"received": {
							Type:        "boolean",
							Description: "Whether the webhook was received (false for errors)",
						},
						"error": {
							Type:        "string",
							Description: "Error message describing what went wrong",
						},
					},
					Required: []string{"received", "error"},
				},
			},
		},
	}
}

// addStandardErrorResponsesForWebhook adds standard error responses for webhooks, excluding 400 which has webhook-specific handling.
func (g *ConventionOpenAPIGenerator) addStandardErrorResponsesForWebhook(operation *Operation, components *Components) {
	// Ensure standard error response components exist
	ensureStdResponses(components)

	// Ensure error response schemas exist in components
	g.ensureErrorSchemas(components)

	// Add standard error responses to the operation
	if operation.Responses == nil {
		operation.Responses = map[string]*Response{}
	}

	// Skip 400 since webhooks have custom 400 handling
	// 422 Unprocessable Entity - Request body could not be parsed
	operation.Responses["422"] = &Response{
		Ref: "#/components/responses/UnprocessableEntity",
	}

	// 500 Internal Server Error - Server error
	operation.Responses["500"] = &Response{
		Ref: "#/components/responses/InternalServerError",
	}
}
