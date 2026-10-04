package api

// SchemaFieldSuffix represents the various field suffixes used in contextual schema naming.
type SchemaFieldSuffix string

// String returns the string representation of the schema field suffix.
func (s SchemaFieldSuffix) String() string {
	return string(s)
}

const (
	// SchemaSuffixBody represents request/response body schemas.
	SchemaSuffixBody SchemaFieldSuffix = "Body"
	// SchemaSuffixHeaders represents request/response header schemas.
	SchemaSuffixHeaders SchemaFieldSuffix = "Headers"
	// SchemaSuffixQuery represents request query parameter schemas.
	SchemaSuffixQuery SchemaFieldSuffix = "Query"
	// SchemaSuffixPath represents request path parameter schemas.
	SchemaSuffixPath SchemaFieldSuffix = "Path"
	// SchemaSuffixCookies represents request/response cookie schemas.
	SchemaSuffixCookies SchemaFieldSuffix = "Cookies"
	// SchemaSuffixResponse represents response schemas.
	SchemaSuffixResponse SchemaFieldSuffix = "Response"
)

// Integration of AST documentation into the runtime-generated OpenAPI spec.

// GenerateOpenAPIWithDocs combines route information from the given registry
// with documentation parsed by DocExtractor to enrich operation and schema
// descriptions. The function delegates the core generation work to
// GenerateOpenAPI and then post-processes the specification.
func GenerateOpenAPIWithDocs(reg *RouteRegistry, extractor *DocExtractor, opts ...OpenAPIOption) *OpenAPISpec {
	spec := GenerateOpenAPI(reg, opts...)
	if extractor == nil {
		return spec
	}
	enrichSpecWithDocs(spec, extractor)
	return spec
}

// enrichSpecWithDocs applies the docs of the Go types that the generator
// recorded for each schema and each operation.
func enrichSpecWithDocs(spec *OpenAPISpec, extractor *DocExtractor) {
	for _, schema := range spec.Components.Schemas {
		enrichSchemaWithTypeDocs(schema, schema.docTypes, extractor)
	}
	enrichPathOperations(spec, extractor)
}

// enrichSchemaWithTypeDocs gives the schema the doc of the first Go type in
// docTypes. Each property gets the field doc of the first Go type that documents it.
func enrichSchemaWithTypeDocs(schema *Schema, docTypes []string, extractor *DocExtractor) {
	if len(docTypes) == 0 {
		return
	}
	if doc := extractor.ExtractTypeDoc(docTypes[0]); doc.Description != "" {
		schema.Description = doc.Description
	}
	for propName, propSchema := range schema.Properties {
		for _, docType := range docTypes {
			if fieldDoc, ok := extractor.ExtractTypeDoc(docType).Fields[propName]; ok {
				propSchema.Description = fieldDoc.Description
				break
			}
		}
	}
}

func enrichPathOperations(spec *OpenAPISpec, extractor *DocExtractor) {
	for _, item := range spec.Paths {
		updateOperationWithDocs(item.Get, extractor)
		updateOperationWithDocs(item.Post, extractor)
		updateOperationWithDocs(item.Put, extractor)
		updateOperationWithDocs(item.Patch, extractor)
		updateOperationWithDocs(item.Delete, extractor)
	}
}

func updateOperationWithDocs(op *Operation, extractor *DocExtractor) {
	if op == nil || extractor == nil {
		return
	}
	doc := extractor.ExtractFunctionDoc(op.OperationID)
	if doc.Description != "" {
		op.Description = doc.Description
	}

	// Enhance parameters with documentation
	enrichParametersWithDocs(op, extractor)
}

// parameterSections maps the "in" value of a parameter to its request section.
var parameterSections = map[string]string{
	"query":  SectionQuery,
	"path":   SectionPath,
	"header": SectionHeaders,
	"cookie": SectionCookies,
}

// enrichParametersWithDocs gives each parameter the field doc from its section of the request type.
func enrichParametersWithDocs(op *Operation, extractor *DocExtractor) {
	for i := range op.Parameters {
		param := &op.Parameters[i]
		section := extractor.ExtractTypeDoc(op.docType + "." + parameterSections[param.In])
		if fieldDoc, ok := section.Fields[param.Name]; ok {
			param.Description = fieldDoc.Description
		}
	}
}
