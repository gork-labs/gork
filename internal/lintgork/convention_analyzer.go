package lintgork

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Standard section names as defined in the Convention Over Configuration spec.
var allowedSections = map[string]bool{
	"Query":   true,
	"Body":    true,
	"Path":    true,
	"Headers": true,
	"Cookies": true,
}

// analyzeConventionStructure analyzes structs for Convention Over Configuration compliance.
func analyzeConventionStructure(file *ast.File, pass *analysis.Pass) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			analyzeHandler(node)
		case *ast.TypeSpec:
			if st, ok := node.Type.(*ast.StructType); ok {
				analyzeRequestStructure(node.Name.Name, st, pass)
			}
		}
		return true
	})
}

// analyzeHandler analyzes handler functions for Convention Over Configuration compliance.
func analyzeHandler(fn *ast.FuncDecl) {
	if fn.Type == nil || len(fn.Type.Params.List) != 2 {
		return // Not a valid handler
	}

	// Check if this looks like a handler function
	if !isHandlerFunction(fn) {
		return
	}

	// Get the request type (second parameter)
	if len(fn.Type.Params.List) >= 2 {
		reqParam := fn.Type.Params.List[1]
		if reqType := extractTypeName(reqParam.Type); reqType != "" {
			// This would need to be linked with struct analysis
			// For now, just validate the handler signature
			validateHandlerSignature(fn)
		}
	}
}

// isHandlerFunction checks if a function looks like a Gork handler.
func isHandlerFunction(fn *ast.FuncDecl) bool {
	if fn.Type == nil {
		return false
	}

	// Check parameters: (context.Context, RequestType)
	if len(fn.Type.Params.List) != 2 {
		return false
	}

	// Check first parameter is context.Context
	firstParam := fn.Type.Params.List[0]
	if !isContextType(firstParam.Type) {
		return false
	}

	// Check return values: (ResponseType, error)
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 {
		return false
	}

	// Check second return value is error
	secondResult := fn.Type.Results.List[1]
	return isErrorType(secondResult.Type)
}

// validateHandlerSignature validates that a handler follows the correct signature.
func validateHandlerSignature(fn *ast.FuncDecl) {
	if fn.Type == nil {
		return
	}

	// Additional validation can be added here
	// For now, this is a placeholder for future enhancements
}

// analyzeRequestStructure analyzes request struct for Convention Over Configuration compliance.
func analyzeRequestStructure(structName string, st *ast.StructType, pass *analysis.Pass) {
	if !isRequestStruct(structName) {
		return
	}

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue // Anonymous field
		}

		fieldName := field.Names[0].Name

		// Check if this is a standard section
		if allowedSections[fieldName] {
			validateConventionSection(fieldName, field, pass)
		}
	}
}

// Reporter interface for dependency injection in testing.
type Reporter interface {
	Reportf(pos token.Pos, format string, args ...interface{})
}

// validateConventionSection validates a Convention Over Configuration section.
func validateConventionSection(sectionName string, field *ast.Field, reporter Reporter) {
	if reporter == nil {
		return
	}

	// Section must be a struct type
	structType, ok := field.Type.(*ast.StructType)
	if !ok {
		reporter.Reportf(field.Pos(), "section '%s' must be a struct type", sectionName)
		return
	}

	// Validate fields within the section
	for _, sectionField := range structType.Fields.List {
		validateSectionField(sectionName, sectionField, reporter)
	}
}

// validateSectionField validates a field within a Convention Over Configuration section.
func validateSectionField(sectionName string, field *ast.Field, reporter Reporter) {
	if reporter == nil {
		return
	}

	if len(field.Names) == 0 {
		return // Anonymous field
	}

	fieldName := field.Names[0].Name

	// Check for gork tag
	if field.Tag == nil {
		reporter.Reportf(field.Pos(), "field '%s.%s' missing gork tag", sectionName, fieldName)
		return
	}

	tagValue := field.Tag.Value
	if !strings.Contains(tagValue, "gork:") {
		reporter.Reportf(field.Pos(), "field '%s.%s' missing gork tag", sectionName, fieldName)
		return
	}

	// Validate gork tag format
	validateGorkTag(sectionName, fieldName, tagValue, field, reporter)
}

// validateGorkTag validates the format of a gork tag.
func validateGorkTag(sectionName, fieldName, tagValue string, field *ast.Field, reporter Reporter) {
	if reporter == nil {
		return
	}

	// Extract gork tag value
	gorkTag := extractGorkTagValue(tagValue)
	if gorkTag == "" {
		reporter.Reportf(field.Pos(), "field '%s.%s' has empty gork tag", sectionName, fieldName)
		return
	}

	// Parse gork tag
	parts := strings.Split(gorkTag, ",")

	wireFormat := strings.TrimSpace(parts[0])
	if wireFormat == "" {
		reporter.Reportf(field.Pos(), "field '%s.%s' gork tag missing wire format name", sectionName, fieldName)
		return
	}

	// Validate options
	for i := 1; i < len(parts); i++ {
		option := strings.TrimSpace(parts[i])
		validateGorkTagOption(sectionName, fieldName, option, field, reporter)
	}
}

// validateGorkTagOption validates a gork tag option.
func validateGorkTagOption(sectionName, fieldName, option string, field *ast.Field, reporter Reporter) {
	if reporter == nil {
		return
	}

	if !strings.Contains(option, "=") {
		reporter.Reportf(field.Pos(), "field '%s.%s' has invalid gork tag option '%s'", sectionName, fieldName, option)
		return
	}

	kv := strings.SplitN(option, "=", 2)

	key := strings.TrimSpace(kv[0])
	value := strings.TrimSpace(kv[1])

	switch key {
	case "discriminator":
		if value == "" {
			reporter.Reportf(field.Pos(), "field '%s.%s' discriminator value cannot be empty", sectionName, fieldName)
		}
	default:
		reporter.Reportf(field.Pos(), "field '%s.%s' unknown gork tag option '%s'", sectionName, fieldName, key)
	}
}

// Helper functions

// isRequestStruct checks if a struct name suggests it's a request struct.
func isRequestStruct(structName string) bool {
	return strings.HasSuffix(structName, "Request")
}

// isResponseStruct checks if a struct name suggests it's a response struct.
func isResponseStruct(structName string) bool {
	return strings.HasSuffix(structName, "Response")
}

// isContextType checks if a type expression represents context.Context.
func isContextType(expr ast.Expr) bool {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "context" && sel.Sel.Name == "Context"
		}
	}
	return false
}

// isErrorType checks if a type expression represents error.
func isErrorType(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "error"
	}
	return false
}

// extractTypeName extracts the type name from a type expression.
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.SelectorExpr:
		return extractTypeName(t.Sel)
	default:
		return ""
	}
}

// extractGorkTagValue extracts the value of a gork tag from a struct tag.
func extractGorkTagValue(tagValue string) string {
	// Remove surrounding backticks
	tagValue = strings.Trim(tagValue, "`")

	// Split by space and look for gork: tag
	parts := strings.Fields(tagValue)
	for _, part := range parts {
		if strings.HasPrefix(part, "gork:") {
			// Extract the value (remove gork: and surrounding quotes)
			value := strings.TrimPrefix(part, "gork:")
			value = strings.Trim(value, `"`)
			return value
		}
	}
	return ""
}
