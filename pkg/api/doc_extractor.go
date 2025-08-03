package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Documentation holds extracted information from Go doc comments.
type Documentation struct {
	Description string
	Fields      map[string]FieldDoc
	Deprecated  bool
	Example     string
	Since       string
}

// FieldDoc represents documentation information for a struct field.
type FieldDoc struct {
	Description string
	Example     string
	Deprecated  bool
}

// DocExtractor parses Go source files and indexes doc comments for later
// lookup by name.
type DocExtractor struct {
	docs map[string]Documentation // fully-qualified name -> documentation
}

// NewDocExtractor allocates a new instance.
func NewDocExtractor() *DocExtractor {
	return &DocExtractor{docs: map[string]Documentation{}}
}

// ParseDirectory walks through the provided directory (recursively) and parses
// every Go file it finds. It ignores vendor directories.
func (d *DocExtractor) ParseDirectory(dir string) error {
	fset := token.NewFileSet()
	// parser.ParseDir does not walk recursively, so we need to walk manually.
	return filepath.WalkDir(dir, func(path string, de os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return d.processDirectoryEntry(path, de, fset)
	})
}

func (d *DocExtractor) processDirectoryEntry(path string, de os.DirEntry, fset *token.FileSet) error {
	if !de.IsDir() {
		return nil
	}
	// Skip vendor
	if de.Name() == "vendor" {
		return filepath.SkipDir
	}

	// Read all Go files in the directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		if err := d.parseFile(filePath, fset); err != nil {
			// Skip files that fail to parse
			continue
		}
	}

	return nil
}

func (d *DocExtractor) parseFile(filePath string, fset *token.FileSet) error {
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(file, d.inspectNode)
	return nil
}

func (d *DocExtractor) inspectNode(n ast.Node) bool {
	switch decl := n.(type) {
	case *ast.GenDecl:
		d.processGenDecl(decl)
	case *ast.FuncDecl:
		d.processFuncDecl(decl)
	}
	return true // continue traversing children
}

func (d *DocExtractor) processGenDecl(decl *ast.GenDecl) {
	if decl.Doc == nil || decl.Tok != token.TYPE {
		return
	}

	for _, spec := range decl.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			d.processTypeSpec(ts, decl.Doc)
		}
	}
}

func (d *DocExtractor) processTypeSpec(ts *ast.TypeSpec, docComment *ast.CommentGroup) {
	name := ts.Name.Name
	// Retrieve or initialize existing doc entry for the type so that we can
	// merge struct-level and field-level information.
	doc := d.docs[name]
	// Top-level type description (paragraph above `type X struct`)
	if docComment != nil {
		doc.Description = extractDescription(docComment.Text())
	}

	// If the underlying type is a struct, iterate over its fields and grab
	// their doc comments. We store them in doc.Fields keyed by the field
	// identifier so that later integration can attach them to schema
	// properties.
	if st, ok := ts.Type.(*ast.StructType); ok {
		d.processStructFields(st, &doc)
	}

	d.docs[name] = doc
}

func (d *DocExtractor) processStructFields(st *ast.StructType, doc *Documentation) {
	if doc.Fields == nil {
		doc.Fields = map[string]FieldDoc{}
	}

	for _, fld := range st.Fields.List {
		desc := d.extractFieldDescription(fld)
		if desc != "" {
			d.storeFieldDocumentation(fld, desc, doc)
		}

		// Also process anonymous struct fields recursively
		d.processAnonymousStructFields(fld, doc)
	}
}

// processAnonymousStructFields recursively processes fields in anonymous structs.
func (d *DocExtractor) processAnonymousStructFields(fld *ast.Field, doc *Documentation) {
	// Check if this field is an anonymous struct (no field names means it's embedded)
	if len(fld.Names) > 0 {
		// This is a named field, check if it's a struct type
		if st, ok := fld.Type.(*ast.StructType); ok {
			// This is a named struct field, process its fields recursively
			d.processStructFields(st, doc)
		}
	}
}

func (d *DocExtractor) extractFieldDescription(fld *ast.Field) string {
	var desc string
	if fld.Doc != nil {
		desc = extractDescription(fld.Doc.Text())
	} else if fld.Comment != nil {
		desc = extractDescription(fld.Comment.Text())
	}
	return desc
}

func (d *DocExtractor) storeFieldDocumentation(fld *ast.Field, desc string, doc *Documentation) {
	for _, ident := range fld.Names {
		// Store by Go identifier
		doc.Fields[ident.Name] = FieldDoc{Description: desc}

		// Also store by JSON tag name if present and differs
		d.storeFieldDocByJSONTag(fld, desc, doc)
	}
}

func (d *DocExtractor) storeFieldDocByJSONTag(fld *ast.Field, desc string, doc *Documentation) {
	if fld.Tag == nil {
		return
	}

	tagVal := strings.Trim(fld.Tag.Value, "`")
	st := reflect.StructTag(tagVal)

	// Check for gork tag
	gorkTag := st.Get("gork")
	if gorkTag != "" {
		if comma := strings.Index(gorkTag, ","); comma != -1 {
			gorkTag = gorkTag[:comma]
		}
		if gorkTag != "" {
			doc.Fields[gorkTag] = FieldDoc{Description: desc}
		}
	}
}

func (d *DocExtractor) processFuncDecl(decl *ast.FuncDecl) {
	if decl.Doc != nil {
		name := decl.Name.Name
		d.docs[name] = Documentation{
			Description: extractDescription(decl.Doc.Text()),
		}
	}
}

// ExtractTypeDoc returns the extracted documentation for the given type name.
func (d *DocExtractor) ExtractTypeDoc(typeName string) Documentation {
	if doc, ok := d.docs[typeName]; ok {
		return doc
	}
	return Documentation{}
}

// ExtractFunctionDoc returns the extracted documentation for the given function name.
func (d *DocExtractor) ExtractFunctionDoc(funcName string) Documentation {
	if doc, ok := d.docs[funcName]; ok {
		return doc
	}
	return Documentation{}
}

// GetAllTypeNames returns all type names that have documentation.
func (d *DocExtractor) GetAllTypeNames() []string {
	var names []string
	for name, doc := range d.docs {
		// Only include types that have field documentation (indicating they're struct types)
		if len(doc.Fields) > 0 {
			names = append(names, name)
		}
	}
	return names
}

// extractDescription returns the first paragraph (until double newline) trimmed.
func extractDescription(comment string) string {
	trimmed := strings.TrimSpace(comment)
	if trimmed == "" {
		return ""
	}

	paragraphs := strings.Split(trimmed, "\n\n")
	// Remove leading comment markers if present
	lines := strings.Split(paragraphs[0], "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(strings.TrimPrefix(l, "//"))
		lines[i] = strings.TrimSpace(strings.TrimPrefix(lines[i], "/*"))
		lines[i] = strings.TrimSpace(strings.TrimSuffix(lines[i], "*/"))
	}
	return strings.TrimSpace(strings.Join(lines, " "))
}
