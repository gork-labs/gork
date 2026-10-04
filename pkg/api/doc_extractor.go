package api

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/mod/modfile"
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
//
// The key of a type is "<import path>.<type name>", for example
// "example.com/app/api.User". A field with an inline struct type adds
// ".<field name>" to the key of its struct, for example
// "example.com/app/api.CreateUserRequest.Body". The key of a function is its name.
type DocExtractor struct {
	docs map[string]Documentation
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
	return fs.WalkDir(os.DirFS(dir), ".", func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return d.processDirectoryEntry(filepath.Join(dir, path), de, fset)
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

	pkgPath, err := importPath(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		if err := d.parseFile(filePath, pkgPath, fset); err != nil {
			// Skip files that fail to parse
			continue
		}
	}

	return nil
}

// importPath returns the import path of the package in dir. It reads the
// module path from the nearest go.mod file in dir or in a parent of dir.
func importPath(dir string) (string, error) {
	abs, _ := filepath.Abs(dir)
	rel := "."
	for {
		data, err := fs.ReadFile(os.DirFS(abs), "go.mod")
		if err == nil {
			return path.Join(modfile.ModulePath(data), rel), nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("no go.mod file in %s or in a parent directory", dir)
		}
		rel = path.Join(filepath.Base(abs), rel)
		abs = parent
	}
}

func (d *DocExtractor) parseFile(filePath, pkgPath string, fset *token.FileSet) error {
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(file, func(n ast.Node) bool {
		return d.inspectNode(n, pkgPath)
	})
	return nil
}

func (d *DocExtractor) inspectNode(n ast.Node, pkgPath string) bool {
	switch decl := n.(type) {
	case *ast.GenDecl:
		d.processGenDecl(decl, pkgPath)
	case *ast.FuncDecl:
		d.processFuncDecl(decl)
	}
	return true // continue traversing children
}

func (d *DocExtractor) processGenDecl(decl *ast.GenDecl, pkgPath string) {
	if decl.Doc == nil || decl.Tok != token.TYPE {
		return
	}

	for _, spec := range decl.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			d.processTypeSpec(ts, decl.Doc, pkgPath)
		}
	}
}

func (d *DocExtractor) processTypeSpec(ts *ast.TypeSpec, docComment *ast.CommentGroup, pkgPath string) {
	key := pkgPath + "." + ts.Name.Name
	doc := Documentation{Description: extractDescription(docComment.Text())}
	if st, ok := ts.Type.(*ast.StructType); ok {
		d.processStructFields(st, key, &doc)
	}
	d.docs[key] = doc
}

func (d *DocExtractor) processStructFields(st *ast.StructType, key string, doc *Documentation) {
	if doc.Fields == nil {
		doc.Fields = map[string]FieldDoc{}
	}

	for _, fld := range st.Fields.List {
		desc := d.extractFieldDescription(fld)
		if desc != "" {
			d.storeFieldDocumentation(fld, desc, doc)
		}

		d.processInlineStructField(fld, key)
	}
}

// processInlineStructField stores the doc of a field with an inline struct
// type, such as the Body section of a request, under "<key>.<field name>".
func (d *DocExtractor) processInlineStructField(fld *ast.Field, key string) {
	st, ok := fld.Type.(*ast.StructType)
	if !ok {
		return
	}
	for _, ident := range fld.Names {
		fieldKey := key + "." + ident.Name
		doc := Documentation{Description: d.extractFieldDescription(fld)}
		d.processStructFields(st, fieldKey, &doc)
		d.docs[fieldKey] = doc
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

	// Also store by json property name if present
	jsonTag := st.Get("json")
	if jsonTag != "" {
		if comma := strings.Index(jsonTag, ","); comma != -1 {
			jsonTag = jsonTag[:comma]
		}
		if jsonTag != "" && jsonTag != "-" {
			doc.Fields[jsonTag] = FieldDoc{Description: desc}
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

// ExtractTypeDoc returns the extracted documentation for the type with the given key.
func (d *DocExtractor) ExtractTypeDoc(key string) Documentation {
	if doc, ok := d.docs[key]; ok {
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
