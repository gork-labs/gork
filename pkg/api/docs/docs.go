package docs

// TypeDocs holds extracted documentation for types and their fields.
type TypeDocs struct {
	Types  map[string]string            // type name -> doc string
	Fields map[string]map[string]string // type name -> field name -> doc
}

// ExtractDocs parses the provided packages and extracts GoDoc comments.
//
// NOTE: This is a stub implementation that returns empty docs so that the rest
// of the system can compile. A full AST-based extractor can be added later.
func ExtractDocs(pkgs []string) (*TypeDocs, error) {
	return &TypeDocs{
		Types:  map[string]string{},
		Fields: map[string]map[string]string{},
	}, nil
}
