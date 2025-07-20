package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
		for _, pkg := range pkgs {
			for fileName, file := range pkg.Files {
				_ = fileName // unused but may become handy
				ast.Inspect(file, func(n ast.Node) bool {
					switch decl := n.(type) {
					case *ast.GenDecl:
						if decl.Doc == nil {
							return true // continue to child nodes
						}
						if decl.Tok == token.TYPE {
							for _, spec := range decl.Specs {
								if ts, ok := spec.(*ast.TypeSpec); ok {
									name := ts.Name.Name
									d.docs[name] = Documentation{
										Description: extractDescription(decl.Doc.Text()),
									}
								}
							}
						}
					case *ast.FuncDecl:
						if decl.Doc != nil {
							name := decl.Name.Name
							d.docs[name] = Documentation{
								Description: extractDescription(decl.Doc.Text()),
							}
						}
					}
					return true // continue traversing children
				})
			}
		}
		return nil
	})
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
