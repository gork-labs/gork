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
		generator := NewConventionOpenAPIGenerator(spec, NewDocExtractor())
		op := generator.buildConventionOperation(route, spec.Components)

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

// ensureStdResponses populates common error responses in components.
func ensureStdResponses(comps *Components) {
	if comps.Responses == nil {
		comps.Responses = map[string]*Response{}
	}

	add := func(name, desc string, schemaRef string) {
		if _, ok := comps.Responses[name]; !ok {
			comps.Responses[name] = &Response{
				Description: desc,
				Content: map[string]*MediaType{
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
	// Use the new refactored schema generator for better testability
	generator := NewSchemaGenerator()
	return generator.GenerateSchema(t, registry, makePointerNullable)
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
	// Create a simple conversion from registry map to Components
	components := &Components{Schemas: registry}

	// Use the convention generator for union schema generation
	generator := NewConventionOpenAPIGenerator(nil, NewDocExtractor())
	u := generator.generateUnionSchema(t, components)

	rawName := t.Name()
	typeName := sanitizeSchemaName(rawName)
	if typeName != "" {
		// Choose a human-friendly unique name (guaranteed non-empty since typeName != "")
		unique := uniqueSchemaNameForType(t, registry)
		registry[unique] = u
		return &Schema{Ref: "#/components/schemas/" + unique}
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
		// Also check the package-prefixed alternative used for collision avoidance
		pkgPref := toPascalCase(lastPathComponent(t.PkgPath()))
		if pkgPref != "" {
			alt := pkgPref + typeName
			if _, ok := registry[alt]; ok {
				return &Schema{Ref: "#/components/schemas/" + alt}
			}
		}
	}
	return nil
}

func buildStructSchema(t reflect.Type, registry map[string]*Schema) *Schema {
	// Use the refactored builder for better testability
	builder := NewStructSchemaBuilder()
	return builder.BuildSchema(t, registry)
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
	if discVal, ok := parseDiscriminator(f.Tag.Get("gork")); ok {
		fieldSchema.Enum = []string{discVal}
	}

	// Parse validation tag
	validateTag := f.Tag.Get("validate")
	if validateTag != "" {
		applyValidationConstraints(fieldSchema, validateTag, f.Type, s, f)
	}

	// Try gork tag first, then fall back to field name
	gorkTag := f.Tag.Get("gork")
	var fieldName string
	if gorkTag != "" {
		fieldName = parseGorkTag(gorkTag).Name
	}
	if fieldName == "" {
		fieldName = f.Name
	}
	s.Properties[fieldName] = fieldSchema
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

// BasicTypeMapper defines the interface for mapping Go types to OpenAPI schemas.
type BasicTypeMapper interface {
	MapType(reflect.Kind) *Schema
}

// defaultBasicTypeMapper implements the default type mapping.
type defaultBasicTypeMapper struct{}

func (m defaultBasicTypeMapper) MapType(kind reflect.Kind) *Schema {
	return mapBasicKind(kind)
}

// mapBasicKind maps Go kinds to OpenAPI schema information.
func mapBasicKind(kind reflect.Kind) *Schema {
	if schema := mapBasicKindDirect(kind); schema != nil {
		return schema
	}
	return mapAdvancedKind(kind)
}

// mapBasicKindDirect handles basic Go types directly.
func mapBasicKindDirect(kind reflect.Kind) *Schema {
	if kind == reflect.String {
		return &Schema{Type: "string"}
	}
	if kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 ||
		kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 {
		return &Schema{Type: "integer"}
	}
	if kind == reflect.Float32 || kind == reflect.Float64 {
		return &Schema{Type: "number"}
	}
	if kind == reflect.Bool {
		return &Schema{Type: "boolean"}
	}
	return nil
}

// mapAdvancedKind handles more complex Go types.
func mapAdvancedKind(kind reflect.Kind) *Schema {
	if schema := mapAdvancedKindDirect(kind); schema != nil {
		return schema
	}
	return &Schema{Type: "object"}
}

// mapAdvancedKindDirect handles advanced Go types that need special mapping.
func mapAdvancedKindDirect(kind reflect.Kind) *Schema {
	if kind == reflect.Uintptr {
		return &Schema{Type: "integer", Description: "Pointer-sized integer"}
	}
	if kind == reflect.Complex64 || kind == reflect.Complex128 {
		return &Schema{Type: "object", Description: "Complex number"}
	}
	if kind == reflect.Chan {
		return &Schema{Type: "object", Description: "Channel"}
	}
	if kind == reflect.Func {
		return &Schema{Type: "object", Description: "Function"}
	}
	if kind == reflect.Interface {
		return &Schema{Type: "object", Description: "Interface"}
	}
	if kind == reflect.Map {
		return &Schema{Type: "object", Description: "Map with dynamic keys"}
	}
	if kind == reflect.UnsafePointer {
		return &Schema{Type: "object", Description: "Unsafe pointer"}
	}
	return nil
}

func buildBasicTypeSchema(t reflect.Type) *Schema {
	mapper := defaultBasicTypeMapper{}
	return mapper.MapType(t.Kind())
}

func buildBasicTypeSchemaWithRegistry(t reflect.Type, registry map[string]*Schema) *Schema {
	if t.Kind() == reflect.Ptr {
		return reflectTypeToSchemaInternal(t.Elem(), registry, true)
	}
	return buildBasicTypeSchema(t)
}

// isUnionType checks if the provided type is one of the generic union wrappers
// defined in pkg/unions.
func isUnionType(t reflect.Type) bool {
	if t == nil {
		return false
	}

	// Dereference pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Must be a struct
	if t.Kind() != reflect.Struct {
		return false
	}

	// Check if it's from the unions package
	pkgPath := t.PkgPath()
	if !strings.HasSuffix(pkgPath, "/unions") {
		return false
	}

	// Check if type name matches Union\d+ pattern (including generics like Union2[T,U])
	typeName := t.Name()
	matched, _ := regexp.MatchString(`^Union\d+(\[.*\])?$`, typeName)
	return matched
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
	// Try gork tag first, then fall back to field name
	gorkTag := sf.Tag.Get("gork")
	var fieldName string
	if gorkTag != "" {
		fieldName = parseGorkTag(gorkTag).Name
	}
	if fieldName == "" {
		fieldName = sf.Name
	}

	// Append if not already present
	for _, r := range parent.Required {
		if r == fieldName {
			return
		}
	}
	parent.Required = append(parent.Required, fieldName)
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

// sanitizeSchemaName converts Go type names containing characters not allowed
// in OpenAPI component keys (e.g. brackets, commas, slashes) into a
// conservative snake-ish representation.
// sanitizeGenericTypeName handles generic type names like "Union2[A,B]".
func sanitizeGenericTypeName(name string) string {
	open := strings.Index(name, "[")
	if open == -1 || !strings.HasSuffix(name, "]") {
		return name
	}

	base := name[:open]
	args := name[open+1 : len(name)-1]
	parts := strings.Split(args, ",")

	for i, p := range parts {
		p = strings.TrimSpace(p)
		p = stripPackagePath(p)
		parts[i] = p
	}

	// Reassemble in a stable, readable form
	return base + "_" + strings.Join(parts, "_")
}

// stripPackagePath removes package path from type name.
func stripPackagePath(typeName string) string {
	if idx := strings.LastIndex(typeName, "."); idx != -1 {
		return typeName[idx+1:]
	}
	return typeName
}

// sanitizeCharacters replaces disallowed characters with underscores.
func sanitizeCharacters(name string) string {
	var b strings.Builder
	for _, r := range name {
		if isAllowedSchemaChar(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// isAllowedSchemaChar checks if a character is allowed in schema names.
func isAllowedSchemaChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '.' || r == '-' || r == '_'
}

func sanitizeSchemaName(n string) string {
	if n == "" {
		return ""
	}

	// Handle generic types first
	n = sanitizeGenericTypeName(n)

	// Replace disallowed characters
	return sanitizeCharacters(n)
}

// lastPathComponent returns the last segment of a slash-separated path.
func lastPathComponent(p string) string {
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	return parts[len(parts)-1]
}

// uniqueSchemaNameForType returns a human-friendly unique component name for a type.
// Preference order:
// 1) Simple type name (sanitized)
// 2) PackageName + TypeName (PascalCase prefix)
// 3) PackageName + TypeName + numeric suffix.
func uniqueSchemaNameForType(t reflect.Type, registry map[string]*Schema) string {
	base := sanitizeSchemaName(t.Name())
	if base == "" {
		return ""
	}
	if _, exists := registry[base]; !exists {
		return base
	}
	pkgPref := toPascalCase(lastPathComponent(t.PkgPath()))
	if pkgPref != "" {
		alt := pkgPref + base
		if _, exists := registry[alt]; !exists {
			return alt
		}
		// As a last resort, append a numeric suffix
		for i := 2; ; i++ {
			candidate := alt + strconv.Itoa(i)
			if _, exists := registry[candidate]; !exists {
				return candidate
			}
		}
	}
	// If there's no package info, fallback to numbered variants of base
	for i := 2; ; i++ {
		candidate := base + strconv.Itoa(i)
		if _, exists := registry[candidate]; !exists {
			return candidate
		}
	}
}
