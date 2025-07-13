package generator

import (
	"strings"
)

// RegisterUnionTypeSchema registers a schema for a union type alias
// This handles cases like: type LoginRequest unions.Union3[EmailLogin, PhoneLogin, OAuthLogin]
func (g *Generator) RegisterUnionTypeSchema(typeName string, unionInfo UnionInfo) {
	// Normalize the type name
	schemaName := g.normalizeTypeName(typeName)
	
	// Check if already registered
	if _, exists := g.spec.Components.Schemas[schemaName]; exists {
		return
	}
	
	// Create the oneOf schema
	schema := &Schema{
		OneOf: make([]*Schema, 0, len(unionInfo.UnionTypes)),
	}
	
	// Add each type to oneOf
	for _, memberType := range unionInfo.UnionTypes {
		memberType = strings.TrimSpace(memberType)
		
		// Handle pointer types
		if strings.HasPrefix(memberType, "*") {
			memberType = strings.TrimPrefix(memberType, "*")
		}
		
		// Generate schema reference
		memberSchemaName := g.normalizeTypeName(memberType)
		schemaRef := &Schema{
			Ref: "#/components/schemas/" + memberSchemaName,
		}
		
		schema.OneOf = append(schema.OneOf, schemaRef)
		
		// Ensure the referenced type is processed
		g.ensureTypeProcessed(memberType)
	}
	
	// Register the schema
	g.spec.Components.Schemas[schemaName] = schema
	
	// Add warnings if needed
	g.addUnionWarnings(unionInfo.UnionTypes)
}

// ProcessHandlerUnionTypes processes union types found in handler signatures
func (g *Generator) ProcessHandlerUnionTypes() {
	// Process all handlers to find union types
	for _, handler := range g.handlerMap {
		// Check request type
		if unionInfo := DetectUnionType(handler.RequestType); unionInfo.IsUnion {
			typeName := strings.TrimPrefix(handler.RequestType, "*")
			g.RegisterUnionTypeSchema(typeName, unionInfo)
		}
		
		// Check response type
		if unionInfo := DetectUnionType(handler.ResponseType); unionInfo.IsUnion {
			typeName := strings.TrimPrefix(handler.ResponseType, "*")
			g.RegisterUnionTypeSchema(typeName, unionInfo)
		}
	}
}

// IsUnionTypeAlias checks if a type is likely a union type alias
// This is a heuristic based on the type name pattern
func IsUnionTypeAlias(t ExtractedType) (bool, UnionInfo) {
	// Check if the type has no fields (likely a type alias)
	if len(t.Fields) == 0 {
		// Check if the type name suggests it's a union
		// This is a simple heuristic - in practice you'd need better detection
		return false, UnionInfo{IsUnion: false}
	}
	
	// Check if the type has a single embedded field that is a union
	if len(t.Fields) == 1 && t.Fields[0].Name == "" {
		// Embedded field
		unionInfo := DetectUnionType(t.Fields[0].Type)
		return unionInfo.IsUnion, unionInfo
	}
	
	return false, UnionInfo{IsUnion: false}
}