package api

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

	// Enrich component schemas first so that operations using $ref automatically
	// pick up descriptions.
	for name, schema := range spec.Components.Schemas {
		doc := extractor.ExtractTypeDoc(name)
		if doc.Description != "" {
			schema.Description = doc.Description
		}
		// Propagate field descriptions to component schema properties.
		if len(doc.Fields) > 0 && schema.Properties != nil {
			for propName, propSchema := range schema.Properties {
				if fd, ok := doc.Fields[propName]; ok {
					if propSchema.Description == "" {
						propSchema.Description = fd.Description
					}
				}
			}
		}
	}

	// Update path operations.
	for _, item := range spec.Paths {
		updateOperationWithDocs(item.Get, extractor)
		updateOperationWithDocs(item.Post, extractor)
		updateOperationWithDocs(item.Put, extractor)
		updateOperationWithDocs(item.Patch, extractor)
		updateOperationWithDocs(item.Delete, extractor)
	}

	return spec
}

func updateOperationWithDocs(op *Operation, extractor *DocExtractor) {
	if op == nil || extractor == nil {
		return
	}
	doc := extractor.ExtractFunctionDoc(op.OperationID)
	if doc.Description != "" {
		op.Description = doc.Description
	}

	// Enhance schemas used by request/response bodies if they are inline (not
	// $ref). For simplicity we only add description when schema is embedded
	// directly.
	if rb := op.RequestBody; rb != nil {
		if mt, ok := rb.Content["application/json"]; ok {
			enhanceInlineSchema(mt.Schema, extractor)
		}
	}
	for _, resp := range op.Responses {
		if mt, ok := resp.Content["application/json"]; ok {
			enhanceInlineSchema(mt.Schema, extractor)
		}
	}
}

func enhanceInlineSchema(sch *Schema, extractor *DocExtractor) {
	if sch == nil || extractor == nil {
		return
	}
	if sch.Ref != "" {
		// Components schemas are already enriched earlier.
		return
	}
	if sch.Type == "object" && sch.Description == "" {
		// Attempt to find matching type by looking up properties pattern; this
		// is best effort and may fail for anonymous structs. Skipping for now.
		return
	}
}

// EnhanceOpenAPISpecWithDocs enriches an already generated specification with
// documentation extracted from source code. It can be used when the spec was
// produced by a separate process (e.g. a runtime export) and therefore we no
// longer have access to the RouteRegistry.
func EnhanceOpenAPISpecWithDocs(spec *OpenAPISpec, extractor *DocExtractor) {
	if spec == nil || extractor == nil {
		return
	}

	// Component schemas
	if spec.Components != nil {
		for name, schema := range spec.Components.Schemas {
			doc := extractor.ExtractTypeDoc(name)
			if doc.Description != "" {
				schema.Description = doc.Description
			}
			// Attach field-level descriptions to schema properties if any.
			if len(doc.Fields) > 0 && schema.Properties != nil {
				for propName, propSchema := range schema.Properties {
					if fd, ok := doc.Fields[propName]; ok {
						if propSchema.Description == "" {
							propSchema.Description = fd.Description
						}
					}
				}
			}
		}
	}

	// Paths & operations
	for _, item := range spec.Paths {
		updateOperationWithDocs(item.Get, extractor)
		updateOperationWithDocs(item.Post, extractor)
		updateOperationWithDocs(item.Put, extractor)
		updateOperationWithDocs(item.Patch, extractor)
		updateOperationWithDocs(item.Delete, extractor)
	}
}
