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

	pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	return d.processPackages(pkgs)
}

func (d *DocExtractor) processPackages(pkgs map[string]*ast.Package) error { //nolint:staticcheck // ast.Package is deprecated but still required by parser.ParseDir
	for _, pkg := range pkgs {
		if err := d.processPackageFiles(pkg.Files); err != nil {
			return err
		}
	}
	return nil
}

func (d *DocExtractor) processPackageFiles(files map[string]*ast.File) error {
	for fileName, file := range files {
		_ = fileName // unused but may become handy
		ast.Inspect(file, d.inspectNode)
	}
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
		if desc == "" {
			continue
		}

		d.storeFieldDocumentation(fld, desc, doc)
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
	jsonTag := st.Get("json")
	if jsonTag == "" {
		return
	}

	if comma := strings.Index(jsonTag, ","); comma != -1 {
		jsonTag = jsonTag[:comma]
	}
	if jsonTag != "" {
		doc.Fields[jsonTag] = FieldDoc{Description: desc}
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

// extractDescription returns the first paragraph (until double newline) trimmed.
func extractDescription(comment string) string {
	paragraphs := strings.Split(strings.TrimSpace(comment), "\n\n")
	if len(paragraphs) > 0 {
		// Remove leading comment markers if present
		lines := strings.Split(paragraphs[0], "\n")
		for i, l := range lines {
			lines[i] = strings.TrimSpace(strings.TrimPrefix(l, "//"))
			lines[i] = strings.TrimSpace(strings.TrimPrefix(lines[i], "/*"))
			lines[i] = strings.TrimSpace(strings.TrimSuffix(lines[i], "*/"))
		}
		return strings.TrimSpace(strings.Join(lines, " "))
	}
	return ""
}
