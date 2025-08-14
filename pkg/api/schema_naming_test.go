package api

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

type fooTestType struct{}

type altFooTestType struct{}

type namedBody struct {
	Token string `gork:"token"`
}

type responseWithNamedBody struct {
	Body namedBody
}

type responseWithAnonBody struct {
	Body struct {
		A int `gork:"a"`
	}
}

func TestUniqueSchemaName_BaseAvailable(t *testing.T) {
	reg := map[string]*Schema{}
	typ := reflect.TypeOf(fooTestType{})
	name := uniqueSchemaNameForType(typ, reg)
	if name != "fooTestType" { // base simple name
		t.Fatalf("expected base name, got %q", name)
	}
}

func TestUniqueSchemaName_PrefixedOnCollision(t *testing.T) {
	reg := map[string]*Schema{"fooTestType": {Type: "object"}}
	typ := reflect.TypeOf(fooTestType{})
	name := uniqueSchemaNameForType(typ, reg)
	if name != "ApifooTestType" {
		t.Fatalf("expected package-prefixed unique name 'ApifooTestType', got %q", name)
	}
}

func TestUniqueSchemaName_SuffixOnDoubleCollision(t *testing.T) {
	reg := map[string]*Schema{
		"fooTestType":    {Type: "object"},
		"ApifooTestType": {Type: "object"},
	}
	typ := reflect.TypeOf(fooTestType{})
	name := uniqueSchemaNameForType(typ, reg)
	if name == "fooTestType" || name == "ApifooTestType" {
		t.Fatalf("expected a suffixed unique name, got %q", name)
	}
	if !strings.HasPrefix(name, "ApifooTestType") {
		t.Fatalf("expected name to start with 'ApifooTestType', got %q", name)
	}
}

func TestUniqueSchemaName_NoPkgNumericFallback(t *testing.T) {
	// Builtin types like int have empty PkgPath -> triggers numeric fallback on base
	reg := map[string]*Schema{"int": {Type: "integer"}}
	name := uniqueSchemaNameForType(reflect.TypeOf(int(0)), reg)
	if name != "int2" {
		t.Fatalf("expected int2, got %q", name)
	}
}

func TestUniqueSchemaName_AnonymousReturnsEmpty(t *testing.T) {
	reg := map[string]*Schema{}
	anon := struct{ A int }{}
	name := uniqueSchemaNameForType(reflect.TypeOf(anon), reg)
	if name != "" {
		t.Fatalf("expected empty name for anonymous type, got %q", name)
	}
}

func TestCheckExistingType_BaseAltAndNone(t *testing.T) {
	reg := map[string]*Schema{"fooTestType": {Type: "object"}}
	if ref := checkExistingType(reflect.TypeOf(fooTestType{}), reg); ref == nil || ref.Ref != "#/components/schemas/fooTestType" {
		t.Fatalf("expected ref to base, got %#v", ref)
	}

	// Only alternative prefixed exists
	reg = map[string]*Schema{"ApifooTestType": {Type: "object"}}
	if ref := checkExistingType(reflect.TypeOf(fooTestType{}), reg); ref == nil || ref.Ref != "#/components/schemas/ApifooTestType" {
		t.Fatalf("expected ref to alternative, got %#v", ref)
	}

	// None exists
	reg = map[string]*Schema{}
	if ref := checkExistingType(reflect.TypeOf(fooTestType{}), reg); ref != nil {
		t.Fatalf("expected nil ref when not found, got %#v", ref)
	}
}

func TestHandleUnionType_UniqueNamingCollision(t *testing.T) {
	reg := map[string]*Schema{"fooTestType": {Type: "object"}}
	// Use a non-union struct; handleUnionType still produces a schema and stores it under a unique name
	ref := handleUnionType(reflect.TypeOf(fooTestType{}), reg)
	if ref == nil || ref.Ref == "" {
		t.Fatalf("expected a ref from handleUnionType, got %#v", ref)
	}
	if _, ok := reg["fooTestType"]; !ok {
		t.Fatalf("expected existing base name to remain present")
	}
	// Ensure a new entry under a different key was added
	added := false
	for k := range reg {
		if k != "fooTestType" {
			added = true
			break
		}
	}
	if !added {
		t.Fatalf("expected a uniquely named schema to be added to the registry")
	}
}

func TestHandleUnionType_AnonymousTypeReturnsInline(t *testing.T) {
	reg := map[string]*Schema{}
	anon := struct{ X int }{}
	s := handleUnionType(reflect.TypeOf(anon), reg)
	if s == nil {
		t.Fatalf("expected a schema for anonymous type")
	}
	if s.Ref != "" {
		t.Fatalf("expected non-ref schema for anonymous type, got ref %q", s.Ref)
	}
	if len(reg) != 0 {
		t.Fatalf("expected registry unchanged for anonymous type, got %v entries", len(reg))
	}
}

func TestDefaultRegistrar_AnonymousType_NoRegistration(t *testing.T) {
	reg := map[string]*Schema{}
	r := &defaultTypeRegistrar{}
	s := &Schema{Type: "object"}
	anon := struct{ A int }{}
	ref := r.RegisterType(reflect.TypeOf(anon), s, reg)
	if ref == nil {
		t.Fatalf("expected non-nil schema")
	}
	if ref.Ref != "" {
		t.Fatalf("expected no ref for anonymous type, got %q", ref.Ref)
	}
	if len(reg) != 0 {
		t.Fatalf("expected no registration in registry for anonymous type, got %v entries", len(reg))
	}
}

func TestRegisterType_CollisionNaming(t *testing.T) {
	reg := map[string]*Schema{"altFooTestType": {Type: "object"}}
	r := &defaultTypeRegistrar{}
	s := &Schema{Type: "object"}
	ref := r.RegisterType(reflect.TypeOf(altFooTestType{}), s, reg)
	if ref == nil || ref.Ref == "" {
		t.Fatalf("expected a ref from registrar, got %#v", ref)
	}
	// Ensure schema was stored under the ref name
	name := ref.Ref[len("#/components/schemas/"):]
	if _, ok := reg[name]; !ok {
		t.Fatalf("expected schema stored under %q, registry: %#v", name, reg)
	}
}

func TestGenerateResponseComponentSchema_AnonymousInline(t *testing.T) {
	components := &Components{Schemas: map[string]*Schema{}}
	// Anonymous response type with Body field
	var resp struct{ Body struct{ X int } }
	g := &ConventionOpenAPIGenerator{}
	s := g.generateResponseComponentSchema(reflect.TypeOf(resp), components)
	if s == nil {
		t.Fatalf("expected schema, got nil")
	}
	if s.Ref != "" {
		t.Fatalf("expected inline schema for anonymous response, got ref %q", s.Ref)
	}
}

func TestGenerateResponseComponentSchema_NamedBodyDirectRef(t *testing.T) {
	components := &Components{Schemas: map[string]*Schema{}}
	g := &ConventionOpenAPIGenerator{spec: &OpenAPISpec{Components: components}, extractor: NewDocExtractor()}
	s := g.generateResponseComponentSchema(reflect.TypeOf(responseWithNamedBody{}), components)
	if s == nil || s.Ref == "" {
		t.Fatalf("expected ref schema, got %#v", s)
	}
	if s.Ref != "#/components/schemas/namedBody" {
		t.Fatalf("expected ref to namedBody component, got %q", s.Ref)
	}
}

func TestGenerateResponseComponentSchema_ExistingComponentEarlyReturn(t *testing.T) {
	// If component with the same response name already exists, it should early-return a ref
	type sampleResp struct{ Body struct{ A int } }
	components := &Components{Schemas: map[string]*Schema{"sampleResp": {Type: "object"}}}
	g := &ConventionOpenAPIGenerator{}
	s := g.generateResponseComponentSchema(reflect.TypeOf(sampleResp{}), components)
	if s == nil || s.Ref != "#/components/schemas/sampleResp" {
		t.Fatalf("expected early ref to existing component, got %#v", s)
	}
}

func TestGenerateResponseComponentSchema_CreateComponentFromAnonBody(t *testing.T) {
	components := &Components{Schemas: map[string]*Schema{}}
	g := &ConventionOpenAPIGenerator{spec: &OpenAPISpec{Components: components}, extractor: NewDocExtractor()}
	s := g.generateResponseComponentSchema(reflect.TypeOf(responseWithAnonBody{}), components)
	if s == nil || s.Ref == "" {
		t.Fatalf("expected ref schema, got %#v", s)
	}
	name := strings.TrimPrefix(s.Ref, "#/components/schemas/")
	stored, ok := components.Schemas[name]
	if !ok || stored == nil {
		t.Fatalf("expected stored component %q, not found", name)
	}
	if stored.Properties == nil || stored.Properties["a"] == nil {
		t.Fatalf("expected extracted properties from anon body, got %#v", stored.Properties)
	}
}

func TestHandleUnionType_BaseNameStoredWhenNoCollision(t *testing.T) {
	reg := map[string]*Schema{}
	ref := handleUnionType(reflect.TypeOf(fooTestType{}), reg)
	if ref == nil || ref.Ref != "#/components/schemas/fooTestType" {
		t.Fatalf("expected ref to base name, got %#v", ref)
	}
	if _, ok := reg["fooTestType"]; !ok {
		t.Fatalf("expected registry to contain base name entry")
	}
}

func TestGenerateResponseComponentSchema_CollisionKeepsExisting(t *testing.T) {
	// Pre-seed a component with the same name as the response type; generator should return a ref to it
	type sampleCollisionResp struct{ Body struct{ A int } }
	components := &Components{Schemas: map[string]*Schema{"sampleCollisionResp": {Type: "object"}}}
	g := &ConventionOpenAPIGenerator{spec: &OpenAPISpec{Components: components}, extractor: NewDocExtractor()}
	s := g.generateResponseComponentSchema(reflect.TypeOf(sampleCollisionResp{}), components)
	if s == nil || s.Ref != "#/components/schemas/sampleCollisionResp" {
		t.Fatalf("expected ref to existing component, got %#v", s)
	}
}

func TestGenerateResponseComponentSchema_UnionBodyBypass(t *testing.T) {
	components := &Components{Schemas: map[string]*Schema{}}
	g := &ConventionOpenAPIGenerator{spec: &OpenAPISpec{Components: components}, extractor: NewDocExtractor()}
	// Define a response whose Body is a unions.Union2 of two named structs
	// Using local named structs to keep it simple
	type A struct {
		V string `gork:"v"`
	}
	type B struct {
		X int `gork:"x"`
	}
	type UBody = struct{ Body unions.Union2[A, B] }
	s := g.generateResponseComponentSchema(reflect.TypeOf(UBody{}), components)
	if s == nil {
		t.Fatalf("expected schema for union body")
	}
	// For union body, generator returns schema from body type directly (no wrapper ref)
	if s.Ref != "" && !strings.Contains(s.Ref, "Union2") {
		// Accept either inline union schema or ref to union component depending on generator behavior
		t.Fatalf("expected inline or ref to union schema, got %#v", s)
	}
}

func TestGenerateResponseComponentSchema_BypassWrapperForNamedBody(t *testing.T) {
	// Response with Body being a named struct -> should return ref to body type component, not wrapper
	components := &Components{Schemas: map[string]*Schema{}}
	g := &ConventionOpenAPIGenerator{spec: &OpenAPISpec{Components: components}, extractor: NewDocExtractor()}
	s := g.generateResponseComponentSchema(reflect.TypeOf(responseWithNamedBody{}), components)
	if s == nil || s.Ref != "#/components/schemas/namedBody" {
		t.Fatalf("expected direct ref to namedBody, got %#v", s)
	}
}

func TestRegisterType_BaseNameWhenAvailable(t *testing.T) {
	reg := map[string]*Schema{}
	r := &defaultTypeRegistrar{}
	s := &Schema{Type: "object"}
	ref := r.RegisterType(reflect.TypeOf(fooTestType{}), s, reg)
	if ref == nil || ref.Ref != "#/components/schemas/fooTestType" {
		t.Fatalf("expected ref to base name, got %#v", ref)
	}
	if _, ok := reg["fooTestType"]; !ok {
		t.Fatalf("expected base name registration present")
	}
}

func TestGenerateResponseComponentSchema_NoBodyFieldCreatesEmptyComponent(t *testing.T) {
	// Response type with no Body field
	type respNoBody struct{ Headers struct{ X string } }
	components := &Components{Schemas: map[string]*Schema{}}
	g := &ConventionOpenAPIGenerator{spec: &OpenAPISpec{Components: components}, extractor: NewDocExtractor()}
	s := g.generateResponseComponentSchema(reflect.TypeOf(respNoBody{}), components)
	if s == nil || s.Ref == "" {
		t.Fatalf("expected ref schema, got %#v", s)
	}
	name := strings.TrimPrefix(s.Ref, "#/components/schemas/")
	stored := components.Schemas[name]
	if stored == nil {
		t.Fatalf("expected stored component %q", name)
	}
	if len(stored.Properties) != 0 {
		t.Fatalf("expected no properties for empty component, got %#v", stored.Properties)
	}
}
