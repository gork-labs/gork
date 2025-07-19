package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// Extractor handles extracting types and handlers from Go source
type Extractor struct {
	fileSet  *token.FileSet
	files    map[string]*ast.File  // filepath -> AST
}

// NewExtractor creates a new extractor
func NewExtractor() *Extractor {
	return &Extractor{
		fileSet:  token.NewFileSet(),
		files:    make(map[string]*ast.File),
	}
}

// ParseDirectory parses all Go files in a directory
func (e *Extractor) ParseDirectory(dir string) error {
	pkgs, err := parser.ParseDir(e.fileSet, dir, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse directory %s: %w", dir, err)
	}
	
	for _, pkg := range pkgs {
		// Store individual files
		for filePath, file := range pkg.Files {
			e.files[filePath] = file
		}
	}
	
	return nil
}

// GetFiles returns all parsed files
func (e *Extractor) GetFiles() map[string]*ast.File {
	return e.files
}

// extractPackageName extracts the package name from an AST file
func extractPackageName(file *ast.File) string {
	if file.Name != nil {
		return file.Name.Name
	}
	return ""
}

// ExtractTypes extracts all struct types from parsed packages
func (e *Extractor) ExtractTypes() []ExtractedType {
	var types []ExtractedType
	
	for filePath, file := range e.files {
		pkgName := extractPackageName(file)
		ast.Inspect(file, func(node ast.Node) bool {
			genDecl, ok := node.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				return true
			}
			
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				
				// Handle struct types
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					e.extractStructType(typeSpec, structType, pkgName, genDecl, filePath, &types)
					continue
				}
				
				// Handle type aliases (including union type aliases)
				e.extractTypeAlias(typeSpec, pkgName, genDecl, filePath, &types)
			}
			
			return true
		})
	}
	
	// Extract enum values for type aliases
	e.extractEnumValues(&types)
	
	return types
}

// ExtractHandlers extracts handler functions matching the expected signature
func (e *Extractor) ExtractHandlers() []ExtractedHandler {
	var handlers []ExtractedHandler
	
	for _, file := range e.files {
		pkgName := extractPackageName(file)
		ast.Inspect(file, func(node ast.Node) bool {
				funcDecl, ok := node.(*ast.FuncDecl)
				if !ok {
					return true
				}
				
				if !ast.IsExported(funcDecl.Name.Name) {
					return true
				}
				
				// Check function signature: func(context.Context, RequestType) (ResponseType, error) or func(context.Context, RequestType) error
				if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 2 {
					return true
				}
				
				// Support both signatures: (ResponseType, error) or just (error)
				if funcDecl.Type.Results == nil {
					return true
				}
				numResults := len(funcDecl.Type.Results.List)
				if numResults != 1 && numResults != 2 {
					return true
				}
				
				// Check first param is context.Context
				firstParam := funcDecl.Type.Params.List[0]
				if !e.isContextType(firstParam.Type) {
					return true
				}
				
				// Check last result is error
				lastResult := funcDecl.Type.Results.List[numResults-1]
				if !e.isErrorType(lastResult.Type) {
					return true
				}
				
				// Extract request and response types
				requestType := e.typeString(funcDecl.Type.Params.List[1].Type)
				responseType := ""
				if numResults == 2 {
					// Has response type
					responseType = e.typeString(funcDecl.Type.Results.List[0].Type)
				}
				
				// Extract function comment
				comment := ""
				if funcDecl.Doc != nil {
					comment = strings.TrimSpace(funcDecl.Doc.Text())
				}
				
				handler := ExtractedHandler{
					Name:         funcDecl.Name.Name,
					Package:      pkgName,
					Description:  comment,
					RequestType:  requestType,
					ResponseType: responseType,
				}
				
				handlers = append(handlers, handler)
				
			return true
		})
	}
	
	return handlers
}

// Helper methods

func (e *Extractor) extractFieldComment(field *ast.Field) string {
	if field.Doc != nil {
		return strings.TrimSpace(field.Doc.Text())
	}
	if field.Comment != nil {
		return strings.TrimSpace(field.Comment.Text())
	}
	return ""
}

func (e *Extractor) typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return e.typeString(t.X)
	case *ast.ArrayType:
		return "[]" + e.typeString(t.Elt)
	case *ast.SelectorExpr:
		return e.typeString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + e.typeString(t.Key) + "]" + e.typeString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.IndexExpr:
		// Handle generic types with single parameter
		base := e.typeString(t.X)
		index := e.typeString(t.Index)
		return base + "[" + index + "]"
	case *ast.IndexListExpr:
		// Handle multiple type parameters
		base := e.typeString(t.X)
		var params []string
		for _, idx := range t.Indices {
			params = append(params, e.typeString(idx))
		}
		return base + "[" + strings.Join(params, ", ") + "]"
	case *ast.StructType:
		// For anonymous structs, we'll return a structured representation
		// that can be parsed later
		return e.structTypeString(t)
	default:
		return "unknown"
	}
}

func (e *Extractor) structTypeString(st *ast.StructType) string {
	// Build a string representation of the struct that can be parsed later
	var sb strings.Builder
	sb.WriteString("struct{")
	
	first := true
	for _, field := range st.Fields.List {
		if field.Names == nil {
			continue // Skip embedded fields
		}
		
		for _, name := range field.Names {
			if !first {
				sb.WriteString("; ")
			}
			first = false
			
			sb.WriteString(name.Name)
			sb.WriteString(" ")
			sb.WriteString(e.typeString(field.Type))
			
			// Add the tag if present
			if field.Tag != nil {
				sb.WriteString(" ")
				sb.WriteString(field.Tag.Value)
			}
		}
	}
	
	sb.WriteString("}")
	return sb.String()
}

func (e *Extractor) isPointer(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func (e *Extractor) isContextType(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	
	pkg, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	
	return pkg.Name == "context" && selector.Sel.Name == "Context"
}

func (e *Extractor) isErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "error"
}

// extractStructType extracts struct type information
func (e *Extractor) extractStructType(typeSpec *ast.TypeSpec, structType *ast.StructType, pkgName string, genDecl *ast.GenDecl, filePath string, types *[]ExtractedType) {
	// Extract type comment
	typeComment := ""
	if genDecl.Doc != nil {
		typeComment = strings.TrimSpace(genDecl.Doc.Text())
	} else if typeSpec.Doc != nil {
		typeComment = strings.TrimSpace(typeSpec.Doc.Text())
	} else if typeSpec.Comment != nil {
		typeComment = strings.TrimSpace(typeSpec.Comment.Text())
	}
	
	extracted := ExtractedType{
		Name:        typeSpec.Name.Name,
		Package:     pkgName,
		Description: typeComment,
		SourceFile:  filePath,
	}
	
	// Extract fields
	for _, field := range structType.Fields.List {
		if field.Names == nil {
			// Embedded field
			if embeddedType := e.typeString(field.Type); embeddedType != "" {
				extracted.EmbeddedTypes = append(extracted.EmbeddedTypes, embeddedType)
			}
			continue
		}
		
		for _, name := range field.Names {
			if !ast.IsExported(name.Name) {
				continue
			}
			
			ef := ExtractedField{
				Name:        name.Name,
				Type:        e.typeString(field.Type),
				Description: e.extractFieldComment(field),
				IsPointer:   e.isPointer(field.Type),
			}
			
			// Parse struct tags
			if field.Tag != nil {
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				
				// JSON tag
				if jsonTag, ok := tag.Lookup("json"); ok {
					parts := strings.Split(jsonTag, ",")
					ef.JSONTag = parts[0]
					if ef.JSONTag == "-" {
						continue // skip field
					}
				}
				
				// Validate tag
				if validateTag, ok := tag.Lookup("validate"); ok {
					ef.ValidateTags = validateTag
				}
				
				// OpenAPI tag
				if openapiTag, ok := tag.Lookup("openapi"); ok {
					ef.OpenAPITag = openapiTag
				}
			}
			
			extracted.Fields = append(extracted.Fields, ef)
		}
	}
	
	if len(extracted.Fields) > 0 || extracted.Name != "" {
		*types = append(*types, extracted)
	}
}

// extractTypeAlias extracts type alias information
func (e *Extractor) extractTypeAlias(typeSpec *ast.TypeSpec, pkgName string, genDecl *ast.GenDecl, filePath string, types *[]ExtractedType) {
	// Get the aliased type as a string
	aliasedType := e.typeString(typeSpec.Type)
	
	// Extract type comment
	typeComment := ""
	if genDecl.Doc != nil {
		typeComment = strings.TrimSpace(genDecl.Doc.Text())
	} else if typeSpec.Doc != nil {
		typeComment = strings.TrimSpace(typeSpec.Doc.Text())
	} else if typeSpec.Comment != nil {
		typeComment = strings.TrimSpace(typeSpec.Comment.Text())
	}
	
	// Check if this is a union type alias
	unionInfo := DetectUnionType(aliasedType)
	
	// Determine the base type for simple aliases
	baseType := ""
	if !unionInfo.IsUnion {
		// For simple type aliases, extract the base type
		if ident, ok := typeSpec.Type.(*ast.Ident); ok {
			baseType = ident.Name
		}
	}
	
	// Create an ExtractedType that represents the type alias
	extracted := ExtractedType{
		Name:        typeSpec.Name.Name,
		Package:     pkgName,
		Description: typeComment,
		IsUnionAlias: unionInfo.IsUnion,
		UnionInfo:   unionInfo,
		IsTypeAlias: true,
		AliasedType: aliasedType,
		BaseType:    baseType,
		TypeDef:     aliasedType,
		SourceFile:  filePath,
	}
	
	*types = append(*types, extracted)
}

// extractEnumValues finds constant values for type aliases
func (e *Extractor) extractEnumValues(types *[]ExtractedType) {
	// Create a map of type aliases for quick lookup
	typeAliasMap := make(map[string]*ExtractedType)
	for i := range *types {
		t := &(*types)[i]
		if t.IsTypeAlias && !t.IsUnionAlias {
			typeAliasMap[t.Name] = t
		}
	}
	
	// Look for constants in all files
	for _, file := range e.files {
		ast.Inspect(file, func(node ast.Node) bool {
				genDecl, ok := node.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.CONST {
					return true
				}
				
				// Process each constant spec
				for _, spec := range genDecl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					
					// Check if the constant has a type
					if valueSpec.Type != nil {
						if ident, ok := valueSpec.Type.(*ast.Ident); ok {
							// Check if this type is one of our type aliases
							if typeAlias, exists := typeAliasMap[ident.Name]; exists {
								// Extract the constant value
								for i := range valueSpec.Names {
									if i < len(valueSpec.Values) {
										if value := e.extractConstValue(valueSpec.Values[i]); value != "" {
											typeAlias.EnumValues = append(typeAlias.EnumValues, value)
										}
									}
								}
							}
						}
					}
				}
				
			return true
		})
	}
}

// extractConstValue extracts the string value from a constant expression
func (e *Extractor) extractConstValue(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			// Remove quotes from string literal
			return strings.Trim(v.Value, `"`)
		}
		return v.Value
	default:
		// For now, only handle basic literals
		return ""
	}
}