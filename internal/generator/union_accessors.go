package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// UnionAccessorGenerator generates accessor methods for user-defined union types
type UnionAccessorGenerator struct {
	packageName string
	imports     map[string]string
}

// NewUnionAccessorGenerator creates a new union accessor generator
func NewUnionAccessorGenerator(packageName string) *UnionAccessorGenerator {
	return &UnionAccessorGenerator{
		packageName: packageName,
		imports:     make(map[string]string),
	}
}

// UserDefinedUnion represents a user-defined type alias for a union
type UserDefinedUnion struct {
	TypeName     string   // e.g., "PaymentMethodRequest"
	UnionType    string   // e.g., "unions.Union2[BankPaymentMethod, CreditCardPaymentMethod]"
	UnionSize    int      // 2, 3, or 4
	OptionTypes  []string // ["BankPaymentMethod", "CreditCardPaymentMethod"]
	OptionNames  []string // ["BankPaymentMethod", "CreditCardPaymentMethod"] or custom names
	PackageName  string
}

// GenerateAccessors generates accessor methods for a user-defined union type
func (g *UnionAccessorGenerator) GenerateAccessors(union UserDefinedUnion) (string, error) {
	// Template for accessor methods
	const accessorTemplate = `
{{- range $i, $option := .Options }}
// Is{{$option.Name}} returns true if the union contains {{$option.TypeName}}
func (u *{{$.TypeName}}) Is{{$option.Name}}() bool {
	{{- if eq $.UnionSize 2 }}
	{{- if eq $i 0 }}
	return u.A != nil
	{{- else }}
	return u.B != nil
	{{- end }}
	{{- else if eq $.UnionSize 3 }}
	{{- if eq $i 0 }}
	return u.A != nil
	{{- else if eq $i 1 }}
	return u.B != nil
	{{- else }}
	return u.C != nil
	{{- end }}
	{{- else if eq $.UnionSize 4 }}
	{{- if eq $i 0 }}
	return u.A != nil
	{{- else if eq $i 1 }}
	return u.B != nil
	{{- else if eq $i 2 }}
	return u.C != nil
	{{- else }}
	return u.D != nil
	{{- end }}
	{{- end }}
}

// {{$option.Name}} returns the {{$option.TypeName}} value if present, nil otherwise
func (u *{{$.TypeName}}) {{$option.Name}}() *{{$option.TypeName}} {
	{{- if eq $.UnionSize 2 }}
	{{- if eq $i 0 }}
	return u.A
	{{- else }}
	return u.B
	{{- end }}
	{{- else if eq $.UnionSize 3 }}
	{{- if eq $i 0 }}
	return u.A
	{{- else if eq $i 1 }}
	return u.B
	{{- else }}
	return u.C
	{{- end }}
	{{- else if eq $.UnionSize 4 }}
	{{- if eq $i 0 }}
	return u.A
	{{- else if eq $i 1 }}
	return u.B
	{{- else if eq $i 2 }}
	return u.C
	{{- else }}
	return u.D
	{{- end }}
	{{- end }}
}
{{- end }}

// Value returns the non-nil value from the union
func (u *{{.TypeName}}) Value() interface{} {
	{{- range $i, $option := .Options }}
	{{- if eq $.UnionSize 2 }}
	{{- if eq $i 0 }}
	if u.A != nil {
		return u.A
	}
	{{- else }}
	if u.B != nil {
		return u.B
	}
	{{- end }}
	{{- else if eq $.UnionSize 3 }}
	{{- if eq $i 0 }}
	if u.A != nil {
		return u.A
	}
	{{- else if eq $i 1 }}
	if u.B != nil {
		return u.B
	}
	{{- else }}
	if u.C != nil {
		return u.C
	}
	{{- end }}
	{{- else if eq $.UnionSize 4 }}
	{{- if eq $i 0 }}
	if u.A != nil {
		return u.A
	}
	{{- else if eq $i 1 }}
	if u.B != nil {
		return u.B
	}
	{{- else if eq $i 2 }}
	if u.C != nil {
		return u.C
	}
	{{- else }}
	if u.D != nil {
		return u.D
	}
	{{- end }}
	{{- end }}
	{{- end }}
	return nil
}

{{- range $i, $option := .Options }}

// Set{{$option.Name}} sets the union to contain {{$option.TypeName}}
func (u *{{$.TypeName}}) Set{{$option.Name}}(value *{{$option.TypeName}}) {
	// Clear all fields first
	{{- if eq $.UnionSize 2 }}
	u.A = nil
	u.B = nil
	{{- else if eq $.UnionSize 3 }}
	u.A = nil
	u.B = nil
	u.C = nil
	{{- else if eq $.UnionSize 4 }}
	u.A = nil
	u.B = nil
	u.C = nil
	u.D = nil
	{{- end }}
	
	// Set the appropriate field
	{{- if eq $.UnionSize 2 }}
	{{- if eq $i 0 }}
	u.A = value
	{{- else }}
	u.B = value
	{{- end }}
	{{- else if eq $.UnionSize 3 }}
	{{- if eq $i 0 }}
	u.A = value
	{{- else if eq $i 1 }}
	u.B = value
	{{- else }}
	u.C = value
	{{- end }}
	{{- else if eq $.UnionSize 4 }}
	{{- if eq $i 0 }}
	u.A = value
	{{- else if eq $i 1 }}
	u.B = value
	{{- else if eq $i 2 }}
	u.C = value
	{{- else }}
	u.D = value
	{{- end }}
	{{- end }}
}
{{- end }}
`

	// Prepare template data
	type Option struct {
		Name     string
		TypeName string
	}

	data := struct {
		TypeName  string
		UnionSize int
		Options   []Option
	}{
		TypeName:  union.TypeName,
		UnionSize: union.UnionSize,
		Options:   make([]Option, len(union.OptionTypes)),
	}

	// Generate option names
	for i, optType := range union.OptionTypes {
		// Clean up the type name to create a method name
		name := g.cleanTypeName(optType)
		if len(union.OptionNames) > i && union.OptionNames[i] != "" {
			name = union.OptionNames[i]
		}
		data.Options[i] = Option{
			Name:     name,
			TypeName: optType,
		}
	}

	// Execute template
	tmpl, err := template.New("accessors").Parse(accessorTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// cleanTypeName removes pointer indicators and package prefixes to create a clean method name
func (g *UnionAccessorGenerator) cleanTypeName(typeName string) string {
	// Remove pointer prefix
	typeName = strings.TrimPrefix(typeName, "*")
	
	// Handle slice types
	if strings.HasPrefix(typeName, "[]") {
		typeName = strings.TrimPrefix(typeName, "[]")
		// Remove package prefix if present
		if idx := strings.LastIndex(typeName, "."); idx >= 0 {
			typeName = typeName[idx+1:]
		}
		return typeName + "Slice"
	}
	
	// Handle map types
	if strings.HasPrefix(typeName, "map[") {
		// Extract key and value types
		endIdx := strings.Index(typeName, "]")
		if endIdx > 0 && endIdx < len(typeName)-1 {
			keyType := typeName[4:endIdx]
			valueType := typeName[endIdx+1:]
			// Clean both types
			keyType = g.cleanTypeName(keyType)
			valueType = g.cleanTypeName(valueType)
			return fmt.Sprintf("%sTo%sMap", keyType, valueType)
		}
	}
	
	// Remove package prefix
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		typeName = typeName[idx+1:]
	}
	
	return typeName
}

// GenerateConstructors generates constructor functions for a union type
func (g *UnionAccessorGenerator) GenerateConstructors(union UserDefinedUnion) (string, error) {
	const constructorTemplate = `
{{- range $i, $option := .Options }}
// New{{$.TypeName}}From{{$option.Name}} creates a new {{$.TypeName}} containing {{$option.TypeName}}
func New{{$.TypeName}}From{{$option.Name}}(value *{{$option.TypeName}}) {{$.TypeName}} {
	return {{$.TypeName}}{
		{{- if eq $.UnionSize 2 }}
		{{- if eq $i 0 }}
		A: value,
		{{- else }}
		B: value,
		{{- end }}
		{{- else if eq $.UnionSize 3 }}
		{{- if eq $i 0 }}
		A: value,
		{{- else if eq $i 1 }}
		B: value,
		{{- else }}
		C: value,
		{{- end }}
		{{- else if eq $.UnionSize 4 }}
		{{- if eq $i 0 }}
		A: value,
		{{- else if eq $i 1 }}
		B: value,
		{{- else if eq $i 2 }}
		C: value,
		{{- else }}
		D: value,
		{{- end }}
		{{- end }}
	}
}
{{- end }}
`

	// Prepare template data
	type Option struct {
		Name     string
		TypeName string
	}

	data := struct {
		TypeName  string
		UnionSize int
		Options   []Option
	}{
		TypeName:  union.TypeName,
		UnionSize: union.UnionSize,
		Options:   make([]Option, len(union.OptionTypes)),
	}

	// Generate option names
	for i, optType := range union.OptionTypes {
		name := g.cleanTypeName(optType)
		if len(union.OptionNames) > i && union.OptionNames[i] != "" {
			name = union.OptionNames[i]
		}
		data.Options[i] = Option{
			Name:     name,
			TypeName: optType,
		}
	}

	// Execute template
	tmpl, err := template.New("constructors").Parse(constructorTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse constructor template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute constructor template: %w", err)
	}

	return buf.String(), nil
}

// GenerateFile generates a complete Go file with union accessors
func (g *UnionAccessorGenerator) GenerateFile(unions []UserDefinedUnion, outputPath string) error {
	var buf bytes.Buffer

	// Write package declaration
	fmt.Fprintf(&buf, "// Code generated by openapi-gen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", g.packageName)

	// Generate accessors and constructors for each union
	for _, union := range unions {
		// Generate accessor methods
		accessors, err := g.GenerateAccessors(union)
		if err != nil {
			return fmt.Errorf("failed to generate accessors for %s: %w", union.TypeName, err)
		}
		buf.WriteString(accessors)
		buf.WriteString("\n")

		// Generate constructor functions
		constructors, err := g.GenerateConstructors(union)
		if err != nil {
			return fmt.Errorf("failed to generate constructors for %s: %w", union.TypeName, err)
		}
		buf.WriteString(constructors)
		buf.WriteString("\n")
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted code for debugging
		writeFile(outputPath+".debug", buf.Bytes())
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write to file
	if err := writeFile(outputPath, formatted); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// UserDefinedUnionWithFile represents a user-defined union with its source file
type UserDefinedUnionWithFile struct {
	UserDefinedUnion
	SourceFile string // Path to the source file containing the union type
}

// ExtractUserDefinedUnions finds all user-defined union type aliases in the codebase
func ExtractUserDefinedUnions(types []ExtractedType) []UserDefinedUnion {
	var unions []UserDefinedUnion

	for _, typeInfo := range types {
		// Check if this is a union alias
		if !typeInfo.IsUnionAlias {
			continue
		}

		union := UserDefinedUnion{
			TypeName:    typeInfo.Name,
			UnionType:   typeInfo.TypeDef,
			UnionSize:   typeInfo.UnionInfo.UnionSize,
			OptionTypes: typeInfo.UnionInfo.UnionTypes,
			PackageName: typeInfo.Package,
		}

		unions = append(unions, union)
	}

	return unions
}

// ExtractUserDefinedUnionsWithFiles finds all user-defined union type aliases with their source files
func ExtractUserDefinedUnionsWithFiles(types []ExtractedType) []UserDefinedUnionWithFile {
	var unions []UserDefinedUnionWithFile

	for _, typeInfo := range types {
		// Check if this is a union alias
		if !typeInfo.IsUnionAlias {
			continue
		}

		union := UserDefinedUnionWithFile{
			UserDefinedUnion: UserDefinedUnion{
				TypeName:    typeInfo.Name,
				UnionType:   typeInfo.TypeDef,
				UnionSize:   typeInfo.UnionInfo.UnionSize,
				OptionTypes: typeInfo.UnionInfo.UnionTypes,
				PackageName: typeInfo.Package,
			},
			SourceFile: typeInfo.SourceFile,
		}

		unions = append(unions, union)
	}

	return unions
}

// GenerateColocatedFiles generates accessor files alongside the source files
func (g *UnionAccessorGenerator) GenerateColocatedFiles(unionsWithFiles []UserDefinedUnionWithFile) error {
	// Group unions by source file
	fileUnions := make(map[string][]UserDefinedUnion)
	filePackages := make(map[string]string)
	
	for _, uwf := range unionsWithFiles {
		fileUnions[uwf.SourceFile] = append(fileUnions[uwf.SourceFile], uwf.UserDefinedUnion)
		filePackages[uwf.SourceFile] = uwf.PackageName
	}
	
	// Generate a file for each source file that has unions
	for sourceFile, unions := range fileUnions {
		// Create the output filename by replacing .go with _goapi_gen.go
		outputFile := strings.TrimSuffix(sourceFile, ".go") + "_goapi_gen.go"
		
		// Update generator's package name for this file
		g.packageName = filePackages[sourceFile]
		
		// Generate the file
		if err := g.GenerateFile(unions, outputFile); err != nil {
			return fmt.Errorf("failed to generate accessors for %s: %w", sourceFile, err)
		}
	}
	
	return nil
}

// writeFile writes content to a file, creating directories if necessary
func writeFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, content, 0644)
}