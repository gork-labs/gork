package generator

import (
	"fmt"
	"go/ast"
	"strings"
)

// GenerateUnionSchema creates a oneOf schema for union types
func (g *Generator) GenerateUnionSchema(unionInfo UnionInfo, fieldName string, field ExtractedField) *Schema {
	// Handle Union2, Union3, Union4 types
	schema := &Schema{
		OneOf:       make([]*Schema, 0, len(unionInfo.UnionTypes)),
		Description: field.Description,
	}

	// Check for discriminator in field tags first
	discriminatorField := ""
	if disc := extractDiscriminator(field.OpenAPITag); disc != "" {
		discriminatorField = disc
	}
	
	// Check if union members implement Discriminator interface
	discriminatorMapping := g.detectDiscriminatorImplementations(unionInfo.UnionTypes)
	
	// If we have discriminator implementations, try to detect the field name
	if len(discriminatorMapping) > 0 && discriminatorField == "" {
		discriminatorField = g.detectCommonDiscriminatorField(unionInfo.UnionTypes)
	}
	
	// Set up discriminator if we have both field and mapping
	if discriminatorField != "" && len(discriminatorMapping) > 0 {
		schema.Discriminator = &Discriminator{
			PropertyName: discriminatorField,
			Mapping:      discriminatorMapping,
		}
	}

	// Add each type to oneOf
	for _, typeName := range unionInfo.UnionTypes {
		typeName = strings.TrimSpace(typeName)
		
		// Handle pointer types
		if strings.HasPrefix(typeName, "*") {
			typeName = strings.TrimPrefix(typeName, "*")
		}
		
		// Check if it's an array type
		isArray := false
		if strings.HasPrefix(typeName, "[]") {
			isArray = true
			typeName = strings.TrimPrefix(typeName, "[]")
		}
		
		// Generate schema
		var schemaRef *Schema
		if isArray {
			// For array types, create an array schema with items referencing the element type
			schemaRef = &Schema{
				Type: "array",
				Items: &Schema{
					Ref: "#/components/schemas/" + g.normalizeTypeName(typeName),
				},
			}
		} else {
			// For non-array types, use direct reference
			schemaRef = &Schema{
				Ref: "#/components/schemas/" + g.normalizeTypeName(typeName),
			}
		}
		
		schema.OneOf = append(schema.OneOf, schemaRef)
		
		// Ensure the referenced type is processed
		g.ensureTypeProcessed(typeName)
	}

	// Add warnings if needed
	g.addUnionWarnings(unionInfo.UnionTypes)

	return schema
}


// isRequiredField checks if a field is required based on validator tags
func (g *Generator) isRequiredField(field ExtractedField) bool {
	// Check for "required" in validate tag
	if strings.Contains(field.ValidateTags, "required") {
		return true
	}
	
	// Check for pointer types (non-pointer fields are typically required in JSON)
	if !strings.HasPrefix(field.Type, "*") && field.JSONTag != "" && field.JSONTag != "-" {
		// Non-pointer fields with JSON tags are typically required unless marked omitempty
		return !strings.Contains(field.JSONTag, "omitempty")
	}
	
	return false
}

// getJSONFieldName gets the JSON field name for a field
func (g *Generator) getJSONFieldName(field ExtractedField) string {
	if field.JSONTag == "" || field.JSONTag == "-" {
		return strings.ToLower(field.Name)
	}
	
	// Extract the field name from JSON tag (before any options like omitempty)
	parts := strings.Split(field.JSONTag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	
	return strings.ToLower(field.Name)
}

// normalizeTypeName converts a type name to a schema-friendly name
func (g *Generator) normalizeTypeName(typeName string) string {
	// Remove package qualifiers for schema names
	if idx := strings.LastIndex(typeName, "."); idx > -1 {
		typeName = typeName[idx+1:]
	}
	
	// Handle generic types (should not happen for union members, but just in case)
	if idx := strings.Index(typeName, "["); idx > -1 {
		typeName = typeName[:idx]
	}
	
	return typeName
}

// extractDiscriminator extracts discriminator field name from OpenAPI tag
func extractDiscriminator(openapiTag string) string {
	if openapiTag == "" {
		return ""
	}

	// Parse "discriminator:fieldName" from tag
	parts := strings.Split(openapiTag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "discriminator:") {
			return strings.TrimPrefix(part, "discriminator:")
		}
	}
	return ""
}

// generateDiscriminatorMapping creates the discriminator mapping for union types
func (g *Generator) generateDiscriminatorMapping(types []string, discriminatorField string) map[string]string {
	mapping := make(map[string]string)
	
	for _, typeName := range types {
		// Clean up type name
		typeName = strings.TrimSpace(typeName)
		if strings.HasPrefix(typeName, "*") {
			typeName = strings.TrimPrefix(typeName, "*")
		}
		
		// Normalize the type name for the schema reference
		normalizedName := g.normalizeTypeName(typeName)
		
		// Create mapping key (lowercase version of the type name)
		// You might want to make this configurable or smarter
		key := strings.ToLower(normalizedName)
		
		// Map to the schema reference
		mapping[key] = "#/components/schemas/" + normalizedName
	}
	
	return mapping
}

// addUnionWarnings adds warnings for potential issues with union types
func (g *Generator) addUnionWarnings(unionTypes []string) {
	// Check for potential subset relationships
	for i, typeA := range unionTypes {
		for j, typeB := range unionTypes {
			if i >= j {
				continue
			}
			
			// Clean up type names
			typeA = strings.TrimSpace(strings.TrimPrefix(typeA, "*"))
			typeB = strings.TrimSpace(strings.TrimPrefix(typeB, "*"))
			
			if g.isSubsetType(typeA, typeB) {
				warning := fmt.Sprintf(
					"WARNING: %s appears to be a subset of %s in union. "+
						"Consider adding discriminator fields or reordering union types "+
						"(more specific types should come first).",
					typeA, typeB,
				)
				g.addWarning(warning)
			} else if g.isSubsetType(typeB, typeA) {
				warning := fmt.Sprintf(
					"WARNING: %s appears to be a subset of %s in union. "+
						"Consider adding discriminator fields or reordering union types "+
						"(more specific types should come first).",
					typeB, typeA,
				)
				g.addWarning(warning)
			}
		}
	}
}

// isSubsetType checks if typeA is a subset of typeB based on fields
func (g *Generator) isSubsetType(typeA, typeB string) bool {
	// Normalize type names
	typeA = g.normalizeTypeName(typeA)
	typeB = g.normalizeTypeName(typeB)
	
	// Look up types in our type map
	defA, okA := g.typeMap[typeA]
	defB, okB := g.typeMap[typeB]
	
	if !okA || !okB || typeA == typeB {
		return false
	}
	
	// Check if all fields in A exist in B with compatible types
	for _, fieldA := range defA.Fields {
		if fieldA.JSONTag == "" || fieldA.JSONTag == "-" {
			continue
		}
		
		found := false
		for _, fieldB := range defB.Fields {
			if fieldB.JSONTag == "" || fieldB.JSONTag == "-" {
				continue
			}
			
			if fieldA.JSONTag == fieldB.JSONTag {
				// Found matching field by JSON tag
				// You could add type compatibility checking here
				found = true
				break
			}
		}
		
		if !found {
			return false
		}
	}
	
	// If A has fewer fields than B and all A's fields exist in B, A is a subset
	aFieldCount := countJSONFields(defA.Fields)
	bFieldCount := countJSONFields(defB.Fields)
	
	return aFieldCount > 0 && aFieldCount < bFieldCount
}

// countJSONFields counts fields that will be in JSON
func countJSONFields(fields []ExtractedField) int {
	count := 0
	for _, field := range fields {
		if field.JSONTag != "" && field.JSONTag != "-" {
			count++
		}
	}
	return count
}


// ensureTypeProcessed makes sure a type is processed and added to schemas
func (g *Generator) ensureTypeProcessed(typeName string) {
	// Clean up the type name
	typeName = strings.TrimSpace(strings.TrimPrefix(typeName, "*"))
	normalizedName := g.normalizeTypeName(typeName)
	
	// Check if already processed
	if _, exists := g.spec.Components.Schemas[normalizedName]; exists {
		return
	}
	
	// Try to find the type in our type map
	if typeInfo, ok := g.typeMap[typeName]; ok {
		g.generateSchema(typeInfo)
	} else if typeInfo, ok := g.typeMap[normalizedName]; ok {
		g.generateSchema(typeInfo)
	}
	// If type not found, it might be a built-in or external type
}

// addWarning adds a warning message to be displayed to the user
func (g *Generator) addWarning(warning string) {
	// This assumes you add a warnings field to the Generator struct
	// You'll need to add: warnings []string to the Generator struct
	// For now, just print it
	fmt.Println(warning)
}

// detectDiscriminatorImplementations checks if union member types have methods that
// indicate they implement the Discriminator interface
func (g *Generator) detectDiscriminatorImplementations(types []string) map[string]string {
	mapping := make(map[string]string)
	
	for _, typeName := range types {
		// Clean up type name
		typeName = strings.TrimSpace(typeName)
		if strings.HasPrefix(typeName, "*") {
			typeName = strings.TrimPrefix(typeName, "*")
		}
		
		normalizedName := g.normalizeTypeName(typeName)
		
		// Check if we have metadata about this type implementing Discriminator
		// Look for a method named DiscriminatorValue
		if g.typeImplementsDiscriminator(typeName) {
			// For now, use the type name as the discriminator value
			// In a full implementation, we'd extract the actual value
			// from the method implementation or from a comment/tag
			discriminatorValue := g.getDiscriminatorValue(typeName)
			if discriminatorValue != "" {
				mapping[discriminatorValue] = "#/components/schemas/" + normalizedName
			}
		}
	}
	
	return mapping
}

// typeImplementsDiscriminator checks if a type has a DiscriminatorValue method
func (g *Generator) typeImplementsDiscriminator(typeName string) bool {
	// Check our parsed AST files for the type and its methods
	normalizedName := g.normalizeTypeName(typeName)
	
	// Look through all parsed files
	for _, file := range g.parsedFiles {
		for _, decl := range file.Decls {
			// Check for method declarations
			if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv != nil {
				// Check if this is a method on our type
				if g.isMethodOnType(funcDecl, normalizedName) {
					if funcDecl.Name.Name == "DiscriminatorValue" {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// isMethodOnType checks if a function declaration is a method on the given type
func (g *Generator) isMethodOnType(funcDecl *ast.FuncDecl, typeName string) bool {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return false
	}
	
	recv := funcDecl.Recv.List[0]
	
	// Handle pointer receivers
	switch t := recv.Type.(type) {
	case *ast.Ident:
		return t.Name == typeName
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == typeName
		}
	}
	
	return false
}

// getDiscriminatorValue attempts to extract the discriminator value for a type
func (g *Generator) getDiscriminatorValue(typeName string) string {
	normalizedName := g.normalizeTypeName(typeName)
	
	// Look for the DiscriminatorValue method and check for a comment with the value
	for _, file := range g.parsedFiles {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv != nil {
				if g.isMethodOnType(funcDecl, normalizedName) && funcDecl.Name.Name == "DiscriminatorValue" {
					// Check if there's a comment with the discriminator value
					// Format: // discriminator:"value"
					if funcDecl.Doc != nil {
						for _, comment := range funcDecl.Doc.List {
							if strings.Contains(comment.Text, "discriminator:") {
								parts := strings.Split(comment.Text, "\"")
								if len(parts) >= 2 {
									return parts[1]
								}
							}
						}
					}
				}
			}
		}
	}
	
	// Fallback: use snake_case version of type name
	return camelToSnake(normalizedName)
}

// detectCommonDiscriminatorField looks for common discriminator field names in union types
func (g *Generator) detectCommonDiscriminatorField(types []string) string {
	// Common discriminator field names
	commonNames := []string{"type", "kind", "@type", "_type", "discriminator"}
	
	// Check if all types have one of the common field names
	for _, fieldName := range commonNames {
		if g.allTypesHaveField(types, fieldName) {
			return fieldName
		}
	}
	
	return ""
}

// allTypesHaveField checks if all types have a field with the given JSON name
func (g *Generator) allTypesHaveField(types []string, jsonFieldName string) bool {
	for _, typeName := range types {
		// Clean up type name
		typeName = strings.TrimSpace(strings.TrimPrefix(typeName, "*"))
		normalizedName := g.normalizeTypeName(typeName)
		
		// Look up type definition
		typeDef, ok := g.typeMap[normalizedName]
		if !ok {
			typeDef, ok = g.typeMap[typeName]
			if !ok {
				return false
			}
		}
		
		// Check if type has the field
		hasField := false
		for _, field := range typeDef.Fields {
			if g.getJSONFieldName(field) == jsonFieldName {
				hasField = true
				break
			}
		}
		
		if !hasField {
			return false
		}
	}
	
	return true
}

// camelToSnake converts CamelCase to snake_case
func camelToSnake(s string) string {
	var result []byte
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, byte(strings.ToLower(string(r))[0]))
	}
	return string(result)
}

// ProcessUnionRequestResponse handles union types at handler level
func (g *Generator) ProcessUnionRequestResponse(handler ExtractedHandler) {
	// Check if request type is a union
	if unionInfo := DetectUnionType(handler.RequestType); unionInfo.IsUnion {
		// Ensure all union member types are processed
		for _, uType := range unionInfo.UnionTypes {
			g.ensureTypeProcessed(uType)
		}
	}
	
	// Check if response type is a union
	if unionInfo := DetectUnionType(handler.ResponseType); unionInfo.IsUnion {
		// Ensure all union member types are processed
		for _, uType := range unionInfo.UnionTypes {
			g.ensureTypeProcessed(uType)
		}
	}
}