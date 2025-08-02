package api

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// defaultRouteFilter excludes the internal documentation endpoint (whose
// response type is *OpenAPISpec) from the generated specification.
func defaultRouteFilter(info *RouteInfo) bool {
	if info == nil || info.ResponseType == nil {
		return true
	}

	// Detect *api.OpenAPISpec (exact pointer match)
	specPtrType := reflect.TypeOf(&OpenAPISpec{})
	return info.ResponseType != specPtrType
}

// GenerateOpenAPI converts the runtime RouteRegistry into a basic OpenAPI 3.1
// specification. The implementation focuses on the essential structure needed
// by clients; we will enrich it iteratively.
func GenerateOpenAPI(registry *RouteRegistry, opts ...OpenAPIOption) *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:   "Generated API",
			Version: "0.1.0",
		},
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas: map[string]*Schema{},
		},
	}

	// apply user options
	for _, o := range opts {
		o(spec)
	}

	// Determine active filter (user-provided or default)
	routeFilter := spec.routeFilter
	if routeFilter == nil {
		routeFilter = defaultRouteFilter
	}

	for _, route := range registry.GetRoutes() {
		if !routeFilter(route) {
			continue
		}
		path := normalizePath(route.Path)
		if spec.Paths[path] == nil {
			spec.Paths[path] = &PathItem{}
		}
		op := buildOperation(route, spec.Components)

		// Security mapping
		applySecurityToOperation(route, spec, op)
		attachOperation(spec.Paths[path], strings.ToLower(route.Method), op)
	}

	return spec
}

func applySecurityToOperation(route *RouteInfo, spec *OpenAPISpec, op *Operation) {
	if route.Options == nil || len(route.Options.Security) == 0 {
		return
	}

	if spec.Components.SecuritySchemes == nil {
		spec.Components.SecuritySchemes = map[string]*SecurityScheme{}
	}

	for _, sec := range route.Options.Security {
		var schemeName string
		var scheme SecurityScheme
		switch sec.Type {
		case "basic":
			schemeName = "BasicAuth"
			scheme = SecurityScheme{Type: "http", Scheme: "basic"}
		case "bearer":
			schemeName = "BearerAuth"
			scheme = SecurityScheme{Type: "http", Scheme: "bearer"}
		case "apiKey":
			schemeName = "ApiKeyAuth"
			scheme = SecurityScheme{Type: "apiKey", In: "header", Name: "X-API-Key"}
		default:
			continue
		}

		spec.Components.SecuritySchemes[schemeName] = &scheme
		op.Security = append(op.Security, map[string][]string{schemeName: {}})
	}
}

func normalizePath(p string) string {
	// For stdlib router the path is already of the form "/users/{id}".
	// Placeholder for future conversions.
	return p
}

func attachOperation(item *PathItem, method string, op *Operation) {
	switch method {
	case "get":
		item.Get = op
	case "post":
		item.Post = op
	case "put":
		item.Put = op
	case "patch":
		item.Patch = op
	case "delete":
		item.Delete = op
	}
}

func buildOperation(route *RouteInfo, comps *Components) *Operation {
	operation := &Operation{
		OperationID: route.HandlerName,
	}
	if route.Options != nil {
		operation.Tags = route.Options.Tags
	}

	// Parameters
	operation.Parameters = extractParameters(route.RequestType, comps.Schemas)

	// Auto-add path params not declared in struct
	existing := map[string]struct{}{}
	for _, p := range operation.Parameters {
		if p.In == "path" {
			existing[p.Name] = struct{}{}
		}
	}
	for _, v := range extractPathVars(route.Path) {
		if _, ok := existing[v]; !ok {
			operation.Parameters = append(operation.Parameters, Parameter{
				Name:     v,
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			})
		}
	}

	// Request body
	if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
		schema := reflectTypeToSchema(route.RequestType, comps.Schemas)
		operation.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: schema},
			},
		}
	}

	// 200 response
	respSchema := reflectTypeToSchema(route.ResponseType, comps.Schemas)
	operation.Responses = map[string]*Response{
		"200": {
			Description: "Success",
			Content: map[string]MediaType{
				"application/json": {Schema: respSchema},
			},
		},
		"400": {Ref: "#/components/responses/BadRequest"},
		"422": {Ref: "#/components/responses/UnprocessableEntity"},
		"500": {Ref: "#/components/responses/InternalServerError"},
	}

	// Ensure standard component responses exist
	ensureStdResponses(comps)

	// Ensure error schemas are registered once
	if _, ok := comps.Schemas["ErrorResponse"]; !ok {
		comps.Schemas["ErrorResponse"] = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"error":   {Type: "string"},
				"details": {Type: "object"},
			},
			Required: []string{"error"},
		}
	}
	if _, ok := comps.Schemas["ValidationErrorResponse"]; !ok {
		comps.Schemas["ValidationErrorResponse"] = &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"error":   {Type: "string"},
				"details": {Type: "object"},
			},
			Required: []string{"error"},
		}
	}

	return operation
}

// ensureStdResponses populates common error responses in components.
func ensureStdResponses(comps *Components) {
	if comps.Responses == nil {
		comps.Responses = map[string]*Response{}
	}

	add := func(name, desc string, schemaRef string) {
		if _, ok := comps.Responses[name]; !ok {
			comps.Responses[name] = &Response{
				Description: desc,
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{Ref: schemaRef}},
				},
			}
		}
	}

	add("BadRequest", "Bad Request - Validation failed", "#/components/schemas/ValidationErrorResponse")
	add("UnprocessableEntity", "Unprocessable Entity - Request body could not be parsed", "#/components/schemas/ErrorResponse")
	add("InternalServerError", "Internal Server Error", "#/components/schemas/ErrorResponse")
}

// reflectTypeToSchema converts a Go type into a (very) simple Schema. Complex
// structures such as unions or nested structs are handled recursively but with
// many simplifications.
func reflectTypeToSchema(t reflect.Type, registry map[string]*Schema) *Schema {
	return reflectTypeToSchemaInternal(t, registry, false)
}

// reflectTypeToSchemaInternal is the internal implementation that allows us to control
// whether pointer types should be treated as nullable.
func reflectTypeToSchemaInternal(t reflect.Type, registry map[string]*Schema, makePointerNullable bool) *Schema {
	// Handle pointer types - these are nullable in OpenAPI 3.1 only if makePointerNullable is true
	if t.Kind() == reflect.Ptr {
		if makePointerNullable {
			underlyingSchema := reflectTypeToSchemaInternal(t.Elem(), registry, true)
			return makeNullableSchema(underlyingSchema)
		}
		// For top-level types, just unwrap the pointer without making it nullable
		return reflectTypeToSchemaInternal(t.Elem(), registry, true)
	}

	// Check for built-in or user-defined union types
	if isUnionType(t) || isUnionStruct(t) {
		return handleUnionType(t, registry)
	}

	// Check if this type is already registered
	if schema := checkExistingType(t, registry); schema != nil {
		return schema
	}

	//nolint:exhaustive // Only handle specific types, default case covers the rest
	switch t.Kind() {
	case reflect.Struct:
		return buildStructSchema(t, registry)
	case reflect.Slice, reflect.Array:
		return buildArraySchema(t, registry)
	default:
		return buildBasicTypeSchemaWithRegistry(t, registry)
	}
}

// makeNullableSchema creates a nullable version of the given schema according to OpenAPI 3.1 spec.
// For basic types, it uses the "type": ["actualType", "null"] format.
// For complex types (refs, objects with properties), it uses anyOf with null.
func makeNullableSchema(originalSchema *Schema) *Schema {
	if originalSchema == nil {
		return &Schema{Type: "null"}
	}

	// If it's a reference or has complex properties, use anyOf
	if originalSchema.Ref != "" || originalSchema.Properties != nil || originalSchema.OneOf != nil || originalSchema.AnyOf != nil {
		return &Schema{
			AnyOf: []*Schema{
				originalSchema,
				{Type: "null"},
			},
		}
	}

	// For basic types, use the array format
	if originalSchema.Type != "" {
		return &Schema{
			Types:       []string{originalSchema.Type, "null"},
			Description: originalSchema.Description,
			Title:       originalSchema.Title,
			Minimum:     originalSchema.Minimum,
			Maximum:     originalSchema.Maximum,
			MinLength:   originalSchema.MinLength,
			MaxLength:   originalSchema.MaxLength,
			Pattern:     originalSchema.Pattern,
			Enum:        originalSchema.Enum,
			Items:       originalSchema.Items,
		}
	}

	// Fallback - just add null as anyOf
	return &Schema{
		AnyOf: []*Schema{
			originalSchema,
			{Type: "null"},
		},
	}
}

func handleUnionType(t reflect.Type, registry map[string]*Schema) *Schema {
	u := generateUnionSchema(t, registry)
	rawName := t.Name()
	typeName := sanitizeSchemaName(rawName)
	if typeName != "" {
		registry[typeName] = u
		return &Schema{Ref: "#/components/schemas/" + typeName}
	}
	return u
}

func checkExistingType(t reflect.Type, registry map[string]*Schema) *Schema {
	rawName := t.Name()
	typeName := sanitizeSchemaName(rawName)
	if typeName != "" {
		if _, ok := registry[typeName]; ok {
			return &Schema{Ref: "#/components/schemas/" + typeName}
		}
	}
	return nil
}

func buildStructSchema(t reflect.Type, registry map[string]*Schema) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: map[string]*Schema{},
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}

		// Skip parameter-only fields
		if isOpenAPIParam(f) {
			continue
		}

		// Handle embedded structs
		if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("json") == "" {
			processEmbeddedStruct(f, s, registry)
			continue
		}

		// Process regular field
		processStructField(f, s, registry)
	}

	// Register named types
	rawName := t.Name()
	typeName := sanitizeSchemaName(rawName)
	if typeName != "" {
		s.Title = typeName
		registry[typeName] = s
		return &Schema{Ref: "#/components/schemas/" + typeName}
	}
	return s
}

func processEmbeddedStruct(f reflect.StructField, s *Schema, registry map[string]*Schema) {
	embeddedSchema := reflectTypeToSchemaInternal(f.Type, registry, true)

	// If embeddedSchema is a reference, resolve to actual for property extraction.
	if embeddedSchema.Ref != "" {
		refName := strings.TrimPrefix(embeddedSchema.Ref, "#/components/schemas/")
		if resolved, ok := registry[refName]; ok {
			embeddedSchema = resolved
		}
	}

	if embeddedSchema.Properties != nil {
		for propName, propSchema := range embeddedSchema.Properties {
			s.Properties[propName] = propSchema
		}
	}
	if len(embeddedSchema.Required) > 0 {
		s.Required = append(s.Required, embeddedSchema.Required...)
	}
}

func processStructField(f reflect.StructField, s *Schema, registry map[string]*Schema) {
	fieldSchema := reflectTypeToSchemaInternal(f.Type, registry, true)

	// Handle discriminator values
	if discVal, ok := parseDiscriminator(f.Tag.Get("openapi")); ok {
		fieldSchema.Enum = []string{discVal}
	}

	// Parse validation tag
	validateTag := f.Tag.Get("validate")
	if validateTag != "" {
		applyValidationConstraints(fieldSchema, validateTag, f.Type, s, f)
	}

	jsonName := getFieldJSONName(f)
	s.Properties[jsonName] = fieldSchema
}

func buildArraySchema(t reflect.Type, registry map[string]*Schema) *Schema {
	itemSchema := reflectTypeToSchemaInternal(t.Elem(), registry, true)
	var title, desc string
	// If the element type has a name, expose it for nicer UI rendering.
	if elemName := t.Elem().Name(); elemName != "" {
		title = "[]" + elemName
		desc = "Array of " + elemName
	}
	return &Schema{Title: title, Description: desc, Type: "array", Items: itemSchema}
}

func buildBasicTypeSchema(t reflect.Type) *Schema {
	//nolint:exhaustive // All relevant cases are handled, exhaustive check is incorrect
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Uintptr:
		return &Schema{Type: "integer", Description: "Pointer-sized integer"}
	case reflect.Complex64, reflect.Complex128:
		return &Schema{Type: "object", Description: "Complex number"}
	case reflect.Chan:
		return &Schema{Type: "object", Description: "Channel"}
	case reflect.Func:
		return &Schema{Type: "object", Description: "Function"}
	case reflect.Interface:
		return &Schema{Type: "object", Description: "Interface"}
	case reflect.Map:
		return &Schema{Type: "object", Description: "Map with dynamic keys"}
	case reflect.UnsafePointer:
		return &Schema{Type: "object", Description: "Unsafe pointer"}
	default:
		return &Schema{Type: "object"}
	}
}

func buildBasicTypeSchemaWithRegistry(t reflect.Type, registry map[string]*Schema) *Schema {
	if t.Kind() == reflect.Ptr {
		return reflectTypeToSchemaInternal(t.Elem(), registry, true)
	}
	return buildBasicTypeSchema(t)
}

// generateUnionSchema builds a oneOf schema for unions.UnionX types.
func generateUnionSchema(t reflect.Type, registry map[string]*Schema) *Schema {
	variants, discInfo := extractUnionVariantsAndDiscriminator(t, registry)

	schema := &Schema{OneOf: variants}
	if discInfo.isValid && discInfo.propertyName != "" {
		schema.Discriminator = &Discriminator{
			PropertyName: discInfo.propertyName,
			Mapping:      discInfo.mapping,
		}
	}

	return schema
}

type discriminatorInfo struct {
	propertyName string
	mapping      map[string]string
	isValid      bool
}

func extractUnionVariantsAndDiscriminator(t reflect.Type, registry map[string]*Schema) ([]*Schema, discriminatorInfo) {
	var variants []*Schema
	discInfo := discriminatorInfo{
		mapping: make(map[string]string),
		isValid: true,
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Type.Kind() != reflect.Ptr {
			continue
		}

		vt := f.Type.Elem()
		variants = append(variants, reflectTypeToSchema(vt, registry))

		// Only struct variants can have discriminators
		if discInfo.isValid && vt.Kind() == reflect.Struct {
			processVariantDiscriminator(vt, &discInfo)
		} else {
			discInfo.isValid = false
		}
	}

	return variants, discInfo
}

func processVariantDiscriminator(vt reflect.Type, discInfo *discriminatorInfo) {
	found := false
	for j := 0; j < vt.NumField(); j++ {
		vf := vt.Field(j)
		if tag, ok := vf.Tag.Lookup("openapi"); ok && strings.HasPrefix(tag, "discriminator=") {
			value := strings.TrimPrefix(tag, "discriminator=")
			jsonName := getFieldJSONName(vf)

			if discInfo.propertyName == "" {
				discInfo.propertyName = jsonName
			} else if discInfo.propertyName != jsonName {
				discInfo.isValid = false
				return
			}

			discInfo.mapping[value] = "#/components/schemas/" + vt.Name()
			found = true
			break
		}
	}
	if !found {
		discInfo.isValid = false
	}
}

func getFieldJSONName(f reflect.StructField) string {
	jsonName := f.Tag.Get("json")
	if jsonName == "" {
		jsonName = f.Name
	}
	if comma := strings.Index(jsonName, ","); comma != -1 {
		jsonName = jsonName[:comma]
	}
	return jsonName
}

// isUnionType checks if the provided type is one of the generic union wrappers
// defined in pkg/unions.
func isUnionType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	return t.PkgPath() == "github.com/gork-labs/gork/pkg/unions" && (t.Name() == "Union2" || t.Name() == "Union3" || t.Name() == "Union4")
}

// isUnionStruct checks if the provided type is a user-defined union struct.
// This is a placeholder and would require a more sophisticated check.
func isUnionStruct(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	// Heuristic: exported struct with >=2 pointer fields and no additional
	// metadata. We ignore unexported fields.
	ptrFields := 0
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported field – treat as non-union
			return false
		}
		if f.Type.Kind() != reflect.Ptr {
			return false
		}
		ptrFields++
	}
	return ptrFields >= 2
}

// applyValidationConstraints maps struct tag validation rules into OpenAPI schema fields.
// Supported rules (subset):
//
//	required          -> adds field to parent.Required
//	min / gt / gte    -> minimum / minLength
//	max / lt / lte    -> maximum / maxLength
func applyValidationConstraints(fieldSchema *Schema, validateTag string, fieldType reflect.Type, parent *Schema, sf reflect.StructField) {
	if fieldSchema == nil {
		return
	}

	parts := strings.Split(validateTag, ",")
	for _, p := range parts {
		if p == "required" {
			addRequiredField(parent, sf)
			continue
		}

		key, val := parseValidationRule(p)
		applyValidationRule(fieldSchema, key, val, fieldType)
	}
}

func addRequiredField(parent *Schema, sf reflect.StructField) {
	jsonName := getFieldJSONName(sf)

	// Append if not already present
	for _, r := range parent.Required {
		if r == jsonName {
			return
		}
	}
	parent.Required = append(parent.Required, jsonName)
}

func parseValidationRule(p string) (key, val string) {
	if idx := strings.Index(p, "="); idx != -1 {
		return p[:idx], p[idx+1:]
	}
	return p, ""
}

func applyValidationRule(fieldSchema *Schema, key, val string, fieldType reflect.Type) {
	switch key {
	case "min", "gte", "gt":
		applyMinConstraint(fieldSchema, val, fieldType)
	case "max", "lte", "lt":
		applyMaxConstraint(fieldSchema, val, fieldType)
	case "len":
		applyLenConstraint(fieldSchema, val, fieldType)
	case "regexp":
		fieldSchema.Pattern = val
	case "oneof":
		applyOneOfConstraint(fieldSchema, val)
	}
}

func applyMinConstraint(fieldSchema *Schema, val string, fieldType reflect.Type) {
	if num, err := strconv.ParseFloat(val, 64); err == nil {
		if isStringKind(fieldType) {
			v := int(num)
			fieldSchema.MinLength = &v
		} else {
			fieldSchema.Minimum = &num
		}
	}
}

func applyMaxConstraint(fieldSchema *Schema, val string, fieldType reflect.Type) {
	if num, err := strconv.ParseFloat(val, 64); err == nil {
		if isStringKind(fieldType) {
			v := int(num)
			fieldSchema.MaxLength = &v
		} else {
			fieldSchema.Maximum = &num
		}
	}
}

func applyLenConstraint(fieldSchema *Schema, val string, fieldType reflect.Type) {
	if num, err := strconv.Atoi(val); err == nil {
		if isStringKind(fieldType) {
			fieldSchema.MinLength = &num
			fieldSchema.MaxLength = &num
		}
	}
}

func applyOneOfConstraint(fieldSchema *Schema, val string) {
	opts := strings.Fields(val)
	if len(opts) > 0 {
		fieldSchema.Enum = opts
	}
}

func isStringKind(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.String
}

// parseOpenAPIParam expects a struct tag value of the form "<name>,in=<loc>".
// It returns the extracted name and location (query|path|header) or ok=false if
// the tag does not match this pattern.
func parseOpenAPIParam(tag string) (name, loc string, ok bool) {
	if tag == "" {
		return "", "", false
	}
	parts := strings.Split(tag, ",")
	if len(parts) < 2 {
		return "", "", false
	}
	name = strings.TrimSpace(parts[0])
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "in=") {
			loc = strings.TrimPrefix(p, "in=")
		}
	}
	if name == "" || loc == "" {
		return "", "", false
	}
	return name, loc, true
}

// parseDiscriminator returns the value after "discriminator=" if present.
func parseDiscriminator(tag string) (value string, ok bool) {
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	for _, p := range parts {
		if strings.HasPrefix(p, "discriminator=") {
			return strings.TrimPrefix(p, "discriminator="), true
		}
	}
	return "", false
}

// extractParameters revised to use parseOpenAPIParam.
func extractParameters(t reflect.Type, registry map[string]*Schema) []Parameter {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var params []Parameter
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name, loc, ok := parseOpenAPIParam(f.Tag.Get("openapi"))
		if !ok {
			continue
		}

		param := buildParameterFromField(f, name, loc, registry)
		params = append(params, param)
	}
	return params
}

func buildParameterFromField(f reflect.StructField, name, loc string, registry map[string]*Schema) Parameter {
	schema := reflectTypeToSchema(f.Type, registry)

	// apply enum from oneof validation
	validateTag := f.Tag.Get("validate")
	applyOneOfValidationToSchema(schema, validateTag)

	required := loc == "path" || strings.Contains(validateTag, "required")

	return Parameter{
		Name:     name,
		In:       loc,
		Required: required,
		Schema:   schema,
	}
}

func applyOneOfValidationToSchema(schema *Schema, validateTag string) {
	if strings.HasPrefix(validateTag, "oneof=") || strings.Contains(validateTag, " oneof=") {
		parts := strings.Split(validateTag, "oneof=")
		if len(parts) > 1 {
			enums := strings.Fields(parts[1])
			if len(enums) > 0 {
				schema.Enum = enums
			}
		}
	}
}

// helper to decide if field is parameter.
func isOpenAPIParam(f reflect.StructField) bool {
	_, _, ok := parseOpenAPIParam(f.Tag.Get("openapi"))
	return ok
}

var pathVarRegexp = regexp.MustCompile(`\{([^{}]+)\}`)

func extractPathVars(path string) []string {
	matches := pathVarRegexp.FindAllStringSubmatch(path, -1)
	var vars []string
	for _, m := range matches {
		if len(m) > 1 {
			vars = append(vars, m[1])
		}
	}
	return vars
}

// sanitizeSchemaName converts Go type names containing characters not allowed
// in OpenAPI component keys (e.g. brackets, commas, slashes) into a
// conservative snake-ish representation.
func sanitizeSchemaName(n string) string {
	if n == "" {
		return ""
	}

	// Special handling for instantiated generic types, e.g.
	//   "Union2[github.com/foo.Bar,github.com/foo.Baz]"
	// We want something concise like "Union2_Bar_Baz" without package paths.
	if open := strings.Index(n, "["); open != -1 && strings.HasSuffix(n, "]") {
		base := n[:open]
		args := n[open+1 : len(n)-1]
		parts := strings.Split(args, ",")
		for i, p := range parts {
			p = strings.TrimSpace(p)
			if idx := strings.LastIndex(p, "."); idx != -1 {
				p = p[idx+1:]
			}
			parts[i] = p
		}
		// Reassemble in a stable, readable form.
		n = base + "_" + strings.Join(parts, "_")
	}

	// Replace any remaining disallowed characters with underscores.
	var b strings.Builder
	for _, r := range n {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
