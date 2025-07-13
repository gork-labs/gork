package generator

import (
	"go/ast"
	"reflect"
	"regexp"
	"strings"
)

// Matches Union2[A, B], Union3[A, B, C], Union4[A, B, C, D]
var unionTypeRegex = regexp.MustCompile(`^(?:unions\.)?Union([2-4])\[(.+)\]$`)

// UnionInfo contains information about a detected union type
type UnionInfo struct {
	IsUnion      bool
	UnionSize    int      // 2, 3, or 4 for UnionN
	UnionTypes   []string // The type parameters
	FullTypeName string   // The complete type name
}

// DetectUnionType checks if a type is a Union2, Union3, or Union4
func DetectUnionType(typeName string) UnionInfo {
	// Check for UnionN types
	matches := unionTypeRegex.FindStringSubmatch(typeName)
	if len(matches) < 3 {
		return UnionInfo{IsUnion: false}
	}

	size := matches[1][0] - '0' // Convert '2', '3', '4' to int
	typeParams := matches[2]
	types := parseTypeParameters(typeParams)

	return UnionInfo{
		IsUnion:      true,
		UnionSize:    int(size),
		UnionTypes:   types,
		FullTypeName: typeName,
	}
}

// parseTypeParameters splits generic type parameters considering nested generics
func parseTypeParameters(params string) []string {
	if params == "" {
		return []string{}
	}
	
	var types []string
	var current strings.Builder
	depth := 0

	for _, ch := range params {
		switch ch {
		case '[', '<':
			depth++
			current.WriteRune(ch)
		case ']', '>':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				types = append(types, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		types = append(types, strings.TrimSpace(current.String()))
	}

	return types
}

// IsUnionField checks if a struct field is a union type
func IsUnionField(field *ast.Field) UnionInfo {
	typeName := getTypeName(field.Type)
	return DetectUnionType(typeName)
}

// getTypeName extracts the full type name from an AST expression
func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		// Handle qualified names like unions.Union2
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
	case *ast.IndexExpr:
		// Handle generic types with single parameter like Union2[A]
		base := getTypeName(t.X)
		if base != "" {
			index := getTypeName(t.Index)
			return base + "[" + index + "]"
		}
	case *ast.IndexListExpr:
		// Handle multiple type parameters like Union3[A, B, C]
		base := getTypeName(t.X)
		if base != "" {
			var params []string
			for _, index := range t.Indices {
				params = append(params, getTypeName(index))
			}
			return base + "[" + strings.Join(params, ", ") + "]"
		}
	case *ast.StarExpr:
		// Handle pointer types
		return "*" + getTypeName(t.X)
	case *ast.ArrayType:
		// Handle array types
		return "[]" + getTypeName(t.Elt)
	case *ast.MapType:
		// Handle map types
		key := getTypeName(t.Key)
		value := getTypeName(t.Value)
		return "map[" + key + "]" + value
	}
	return ""
}

// ExtractUnionMemberTypes extracts the actual type names from a union field
// considering package imports and type aliases
func ExtractUnionMemberTypes(unionInfo UnionInfo, imports map[string]string) []string {
	var resolvedTypes []string

	for _, typeName := range unionInfo.UnionTypes {
		// Handle pointer types
		isPointer := false
		if strings.HasPrefix(typeName, "*") {
			isPointer = true
			typeName = strings.TrimPrefix(typeName, "*")
		}

		// Resolve package-qualified types
		if parts := strings.Split(typeName, "."); len(parts) == 2 {
			pkg := parts[0]
			typ := parts[1]
			
			// Look up the actual import path
			if _, ok := imports[pkg]; ok {
				// For now, keep the qualified name
				// In a real implementation, you'd resolve this to the actual type
				typeName = pkg + "." + typ
			}
		}

		if isPointer {
			typeName = "*" + typeName
		}
		
		resolvedTypes = append(resolvedTypes, typeName)
	}

	return resolvedTypes
}

// GetDiscriminatorFromTag extracts discriminator info from struct tags
func GetDiscriminatorFromTag(field *ast.Field) (discriminatorField string, hasDiscriminator bool) {
	if field.Tag == nil {
		return "", false
	}

	tag := strings.Trim(field.Tag.Value, "`")
	tagValue := reflect.StructTag(tag).Get("openapi")
	
	if tagValue == "" {
		return "", false
	}

	// Parse "discriminator:fieldName" from tag
	parts := strings.Split(tagValue, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "discriminator:") {
			return strings.TrimPrefix(part, "discriminator:"), true
		}
	}

	return "", false
}

// IsSubsetType checks if typeA fields are a subset of typeB fields
// This is a simplified version - in practice you'd need the actual type definitions
func IsSubsetType(typeA, typeB string, typeMap map[string]ExtractedType) bool {
	// Remove pointer prefixes for comparison
	typeA = strings.TrimPrefix(typeA, "*")
	typeB = strings.TrimPrefix(typeB, "*")

	// Get type definitions
	defA, okA := typeMap[typeA]
	defB, okB := typeMap[typeB]

	if !okA || !okB {
		return false
	}

	// Check if all fields in A exist in B
	for _, fieldA := range defA.Fields {
		found := false
		for _, fieldB := range defB.Fields {
			if fieldA.Name == fieldB.Name && fieldA.Type == fieldB.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// If A has fewer fields than B and all A's fields exist in B, A is a subset
	return len(defA.Fields) < len(defB.Fields)
}

