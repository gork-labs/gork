package generator

// generateSecuritySchemes generates security scheme definitions based on used security requirements
func (g *Generator) generateSecuritySchemes() {
	usedSchemes := make(map[string]bool)

	// Collect all used security schemes from operations
	for _, pathItem := range g.spec.Paths {
		operations := []*Operation{
			pathItem.Get,
			pathItem.Post,
			pathItem.Put,
			pathItem.Delete,
			pathItem.Patch,
		}

		for _, op := range operations {
			if op != nil && op.Security != nil {
				for _, sec := range op.Security {
					for schemeName := range sec {
						usedSchemes[schemeName] = true
					}
				}
			}
		}
	}

	// Create security scheme definitions for used schemes
	for schemeName := range usedSchemes {
		switch schemeName {
		case "basicAuth":
			g.spec.Components.SecuritySchemes[schemeName] = &SecurityScheme{
				Type:        "http",
				Scheme:      "basic",
				Description: "Basic HTTP authentication",
			}
		case "bearerAuth":
			g.spec.Components.SecuritySchemes[schemeName] = &SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "JWT Bearer token authentication",
			}
		case "apiKeyAuth":
			g.spec.Components.SecuritySchemes[schemeName] = &SecurityScheme{
				Type:        "apiKey",
				Name:        "X-API-Key",
				In:          "header",
				Description: "API key authentication",
			}
		}
	}
}

// convertSecurityRequirements converts internal security requirements to OpenAPI format
func (g *Generator) convertSecurityRequirements(requirements []SecurityRequirement) []map[string][]string {
	var result []map[string][]string

	for _, req := range requirements {
		secMap := make(map[string][]string)

		switch req.Type {
		case "basic":
			secMap["basicAuth"] = []string{}
		case "bearer":
			if req.Scopes == nil {
				secMap["bearerAuth"] = []string{}
			} else {
				secMap["bearerAuth"] = req.Scopes
			}
		case "apiKey":
			secMap["apiKeyAuth"] = []string{}
		}

		if len(secMap) > 0 {
			result = append(result, secMap)
		}
	}

	return result
}
