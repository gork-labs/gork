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
									// Retrieve or initialize existing doc entry for the type so that we can
									// merge struct-level and field-level information.
									doc := d.docs[name]
									// Top-level type description (paragraph above `type X struct`)
									if decl.Doc != nil {
										doc.Description = extractDescription(decl.Doc.Text())
									}

									// If the underlying type is a struct, iterate over its fields and grab
									// their doc comments. We store them in doc.Fields keyed by the field
									// identifier so that later integration can attach them to schema
									// properties.
									if st, ok := ts.Type.(*ast.StructType); ok {
										if doc.Fields == nil {
											doc.Fields = map[string]FieldDoc{}
										}
										for _, fld := range st.Fields.List {
											var desc string
											if fld.Doc != nil {
												desc = extractDescription(fld.Doc.Text())
											} else if fld.Comment != nil {
												desc = extractDescription(fld.Comment.Text())
											}
											if desc == "" {
												continue
											}
											for _, ident := range fld.Names {
												// Store by Go identifier
												doc.Fields[ident.Name] = FieldDoc{Description: desc}

												// Also store by JSON tag name if present and differs
												if fld.Tag != nil {
													tagVal := strings.Trim(fld.Tag.Value, "`")
													st := reflect.StructTag(tagVal)
													jsonTag := st.Get("json")
													if jsonTag != "" {
														if comma := strings.Index(jsonTag, ","); comma != -1 {
															jsonTag = jsonTag[:comma]
														}
														if jsonTag != "" {
															doc.Fields[jsonTag] = FieldDoc{Description: desc}
														}
													}
												}
											}
										}
									}

									d.docs[name] = doc
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
