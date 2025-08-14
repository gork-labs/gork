package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	foo "github.com/gork-labs/gork/pkg/api/testdata/webhooks/foo"
)

// Top-level helpers used by tests
type prov struct{}

func (prov) ProviderInfo() WebhookProviderInfo {
	return WebhookProviderInfo{Name: "FromHandler", Website: "w", DocsURL: "d"}
}

type prov2 struct{}

func (prov2) GetValidEventTypes() []string { return []string{"evt.a", "evt.b"} }

type tReq struct{ Body []byte }

func (tReq) WebhookRequest() {}

type prov3 struct{}

func (prov3) ParseRequest(tReq) (WebhookEvent, error) { return WebhookEvent{}, nil }
func (prov3) SuccessResponse() interface{}            { return foo.WebhookResponse{} }
func (prov3) ErrorResponse(err error) interface{}     { return foo.WebhookErrorResponse{} }
func (prov3) GetValidEventTypes() []string            { return nil }
func (prov3) ProviderInfo() WebhookProviderInfo       { return WebhookProviderInfo{Name: "Api"} }

type respProv struct{}

// proper signatures at package scope
func (respProv) SuccessResponse() interface{}        { return struct{ A int }{1} }
func (respProv) ErrorResponse(err error) interface{} { return struct{ E string }{"e"} }

type wrong struct{}

// wrong signature to test fallback path
func (wrong) ErrorResponse() interface{} { return "x" }

// Handlers for exercising additional branches
type bad0 struct{}

// GetValidEventTypes returns no values -> callEventTypesMethod should return nil
func (bad0) GetValidEventTypes() {}

type badRet struct{}

// GetValidEventTypes has correct arity but wrong return type
func (badRet) GetValidEventTypes() []int { return []int{1, 2} }

// provider info with wrong signatures/returns
type badProv0 struct{}

func (badProv0) ProviderInfo() {} // no return -> should yield nil

type badProv1 struct{}

func (badProv1) ProviderInfo() string { return "nope" } // wrong type -> should yield nil

// Test provider info extraction from route and handler
func TestGetWebhookProviderInfo_Sources(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{}, NewDocExtractor())

	// 1) From route.WebhookProviderInfo
	pi := &WebhookProviderInfo{Name: "Acme", Website: "w", DocsURL: "d"}
	r1 := &RouteInfo{WebhookProviderInfo: pi}
	if got := gen.getWebhookProviderInfo(r1); got == nil || got.Name != "Acme" {
		t.Fatalf("expected provider info from route, got %#v", got)
	}

	// 2) From handler.ProviderInfo() via minimal provider
	r2 := &RouteInfo{WebhookHandler: prov{}}
	got2 := gen.getWebhookProviderInfo(r2)
	if got2 == nil || got2.Name == "" || got2.DocsURL == "" {
		t.Fatalf("expected provider info from handler, got %#v", got2)
	}

	// 3) Nil when no sources
	if got3 := gen.getWebhookProviderInfo(&RouteInfo{}); got3 != nil {
		t.Fatalf("expected nil provider info, got %#v", got3)
	}
}

// Test event type extraction and provider entries
func TestEventTypesAndEntries_FromProvider(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{}, NewDocExtractor())
	route := &RouteInfo{WebhookHandler: prov2{}}

	// extractEventTypesFromHandler -> non-empty
	types := gen.extractEventTypesFromHandler(route.WebhookHandler)
	if len(types) == 0 {
		t.Fatal("expected non-empty event types from handler")
	}

	// buildEventEntriesFromProvider -> entries with {event}
	entries := gen.buildEventEntriesFromProvider(route)
	if len(entries) == 0 || entries[0]["event"] == "" {
		t.Fatalf("expected entries with event, got %#v", entries)
	}
}

// Ensure provider entries fallback to nil when provider advertises none
func TestEventEntriesFromProvider_Empty(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{}, NewDocExtractor())
	route := &RouteInfo{WebhookHandler: prov3{}}
	// prov3 returns nil event types
	if entries := gen.buildEventEntriesFromProvider(route); entries != nil {
		t.Fatalf("expected nil entries when provider returns none, got %#v", entries)
	}
}

// No GetValidEventTypes method present -> extractEventTypesFromHandler returns nil
func TestExtractEventTypes_NoMethod(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	if types := gen.extractEventTypesFromHandler(struct{}{}); types != nil {
		t.Fatalf("expected nil when method missing, got %#v", types)
	}
}

// Invalid GetValidEventTypes signature -> filtered by isValidEventTypesMethod
func TestExtractEventTypes_InvalidSignature(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	if types := gen.extractEventTypesFromHandler(bad0{}); types != nil {
		t.Fatalf("expected nil for invalid signature, got %#v", types)
	}
}

// Test renaming of generic WebhookRequest to provider-specific component
func TestRenameGenericWebhookRequestIfNeeded(t *testing.T) {
	comps := &Components{Schemas: map[string]*Schema{}}
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: comps}, NewDocExtractor())

	reqType := reflect.TypeOf(foo.WebhookRequest{})
	s := gen.buildWebhookRequestBodySchema(reqType, comps)
	if s == nil || s.Ref != "#/components/schemas/FooWebhookRequest" {
		t.Fatalf("expected FooWebhookRequest ref, got %#v", s)
	}
	if _, ok := comps.Schemas["FooWebhookRequest"]; !ok {
		t.Fatalf("expected FooWebhookRequest component to be created")
	}
}

// Test additional paths in renameGenericWebhookRequestIfNeeded
func TestRenameGenericWebhookRequest_ManualSchemaAndEarlyExits(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{}, NewDocExtractor())
	comps := &Components{}
	// Early exit: wrong type name
	rtWrong := reflect.TypeOf(struct{ X int }{})
	if got := gen.renameGenericWebhookRequestIfNeeded(rtWrong, &Schema{Type: "object"}, comps); got == nil || got.Type != "object" {
		t.Fatalf("expected unchanged schema for wrong type, got %#v", got)
	}
	// Manual non-ref schema for proper type -> should create provider-specific component
	reqType := reflect.TypeOf(foo.WebhookRequest{})
	s := &Schema{Type: "object"}
	got := gen.renameGenericWebhookRequestIfNeeded(reqType, s, &Components{Schemas: map[string]*Schema{}})
	if got == nil || got.Ref != "#/components/schemas/FooWebhookRequest" {
		t.Fatalf("expected FooWebhookRequest ref for manual schema, got %#v", got)
	}
}

// Test providerFromPkgPath variations
func TestProviderFromPkgPath(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	if p := gen.providerFromPkgPath("github.com/gork-labs/gork/pkg/webhooks/stripe"); p != "stripe" {
		t.Fatalf("expected stripe, got %s", p)
	}
	if p := gen.providerFromPkgPath(""); p != "" {
		t.Fatalf("expected empty, got %s", p)
	}
	if p := gen.providerFromPkgPath("x/y"); p != "y" {
		t.Fatalf("expected y, got %s", p)
	}
	if p := gen.providerFromPkgPath("a/webhooks"); p != "a" {
		t.Fatalf("expected a (prev segment), got %s", p)
	}
	if p := gen.providerFromPkgPath("webhooks"); p != "webhooks" {
		t.Fatalf("expected webhooks (no provider segment), got %s", p)
	}
}

// Test toPascalCase helper
func TestToPascalCase(t *testing.T) {
	cases := map[string]string{
		"stripe":         "Stripe",
		"stripe_webhook": "StripeWebhook",
		"STRIPE-WEBHOOK": "StripeWebhook",
		"":               "",
		"foo.bar":        "FooBar",
	}
	for in, want := range cases {
		if got := toPascalCase(in); got != want {
			t.Fatalf("toPascalCase(%q)=%q want %q", in, got, want)
		}
	}
}

// Test response schema renaming to provider-specific components
func TestAddWebhookResponses_ProviderSpecificNames(t *testing.T) {
	comps := &Components{Schemas: map[string]*Schema{}}
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: comps}, NewDocExtractor())
	op := &Operation{Responses: map[string]*Response{}}
	route := &RouteInfo{WebhookHandler: prov3{}}

	gen.addWebhookResponses(op, comps, route)

	// 200 response should reference FooWebhookResponse
	r200 := op.Responses["200"]
	if r200 == nil || r200.Content == nil || r200.Content["application/json"].Schema == nil {
		t.Fatal("missing 200 schema")
	}
	if ref := r200.Content["application/json"].Schema.Ref; ref != "#/components/schemas/FooWebhookResponse" {
		t.Fatalf("expected FooWebhookResponse ref, got %s", ref)
	}
	if _, ok := comps.Schemas["FooWebhookResponse"]; !ok {
		t.Fatalf("expected FooWebhookResponse component to be registered")
	}

	// 400 response -> FooWebhookErrorResponse
	r400 := op.Responses["400"]
	if r400 == nil || r400.Content == nil || r400.Content["application/json"].Schema == nil {
		t.Fatal("missing 400 schema")
	}
	if ref := r400.Content["application/json"].Schema.Ref; ref != "#/components/schemas/FooWebhookErrorResponse" {
		t.Fatalf("expected FooWebhookErrorResponse ref, got %s", ref)
	}
}

// Test resolveConcreteType, getErrorResponseType and extractTypeFromResults helpers
func TestReflectionHelpers_ResponseTypes(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)

	v := reflect.ValueOf(respProv{})
	m := v.MethodByName("SuccessResponse")
	rt := gen.resolveConcreteType(m, m.Type(), m.Type().Out(0), "SuccessResponse")
	if rt.Kind() != reflect.Struct {
		t.Fatalf("expected struct type, got %v", rt)
	}

	// wrong signature for ErrorResponse -> returns static return type (interface{})
	w := reflect.ValueOf(wrong{})
	mw := w.MethodByName("ErrorResponse")
	if tp := gen.getErrorResponseType(mw, mw.Type()); tp.Kind() != reflect.Interface {
		t.Fatalf("expected interface return type fallback, got %v", tp)
	}

	// extractTypeFromResults: interface wrapping vs concrete
	s := struct{ Z int }{Z: 5}
	vals := []reflect.Value{reflect.ValueOf(interface{}(s))}
	if t1 := gen.extractTypeFromResults(vals); t1.Kind() != reflect.Struct {
		t.Fatalf("expected struct from interface wrapping, got %v", t1)
	}
	vals2 := []reflect.Value{reflect.ValueOf(42)}
	if t2 := gen.extractTypeFromResults(vals2); t2.Kind() != reflect.Int {
		t.Fatalf("expected int, got %v", t2)
	}
	if t3 := gen.extractTypeFromResults(nil); t3 != nil {
		t.Fatalf("expected nil type, got %v", t3)
	}
}

// Test Operation.MarshalJSON merges Extensions and emits x-* fields
func TestOperationMarshalJSON_Extensions(t *testing.T) {
	op := &Operation{
		OperationID:      "Op",
		Extensions:       map[string]interface{}{"x-foo": map[string]string{"a": "b"}},
		XWebhookProvider: map[string]string{"name": "Stripe"},
		XWebhookEvents:   []map[string]interface{}{{"event": "x"}},
	}
	b, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if m["x-foo"] == nil || m["x-webhook-provider"] == nil || m["x-webhook-events"] == nil {
		t.Fatalf("expected x-* fields in marshaled operation, got %v", m)
	}
}

// Test Operation.MarshalJSON ignores empty extension keys
func TestOperationMarshalJSON_EmptyKeyIgnored(t *testing.T) {
	op := &Operation{OperationID: "Op2", Extensions: map[string]interface{}{"": 123}}
	b, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("marshal err: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m[""]; ok {
		t.Fatalf("empty extension key should be omitted, got %v", m)
	}
}

// Test Operation.MarshalJSON error paths via DI hooks
func TestOperationMarshalJSON_ErrorPaths(t *testing.T) {
	// Marshal error
	saveM := jsonMarshal
	jsonMarshal = func(v interface{}) ([]byte, error) { return nil, errors.New("boom") }
	defer func() { jsonMarshal = saveM }()
	if _, err := (&Operation{}).MarshalJSON(); err == nil {
		t.Fatalf("expected error from marshal hook")
	}

	// Unmarshal error
	saveU := jsonUnmarshal
	jsonMarshal = saveM // restore normal marshal
	jsonUnmarshal = func(data []byte, v interface{}) error { return errors.New("bad") }
	defer func() { jsonUnmarshal = saveU }()
	if _, err := (&Operation{}).MarshalJSON(); err == nil {
		t.Fatalf("expected error from unmarshal hook")
	}
}

// Test deriveWebhookHandlerName helper
func TestDeriveWebhookHandlerName(t *testing.T) {
	// non-webhook route -> http_handler
	if got := deriveWebhookHandlerName(simpleHTTPHandler, false); got != "http_handler" {
		t.Fatalf("expected http_handler, got %s", got)
	}
	// webhook with original handler -> PascalCase provider name + Webhook
	// Create a minimal handler in this package to register and derive name from package path
	httpH := WebhookHandlerFunc[tReq](prov3{})
	if got := deriveWebhookHandlerName(httpH, true); got != "ApiWebhook" {
		t.Fatalf("expected ApiWebhook, got %s", got)
	}
	// webhook with unknown original -> generic webhook_handler
	if got := deriveWebhookHandlerName(simpleHTTPHandler, true); got != "webhook_handler" {
		t.Fatalf("expected webhook_handler, got %s", got)
	}
	// webhook with original pointer type -> covers pointer branch
	httpHP := WebhookHandlerFunc[tReq](&prov3{})
	if got := deriveWebhookHandlerName(httpHP, true); got != "ApiWebhook" {
		t.Fatalf("expected ApiWebhook for pointer original, got %s", got)
	}

	// webhook with original having empty PkgPath but non-empty Name -> name fallback branch
	// Prepare a dummy http.HandlerFunc and manually register a fake original of type int
	dummy := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	registerWebhookHandler(dummy, webhookRegistryEntry{original: int(0)})
	if got := deriveWebhookHandlerName(dummy, true); got != "IntWebhook" {
		t.Fatalf("expected IntWebhook from name fallback, got %s", got)
	}
	// unnamed type: pkg=="", name=="" -> default branch
	dummy2 := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	registerWebhookHandler(dummy2, webhookRegistryEntry{original: struct{}{}})
	if got := deriveWebhookHandlerName(dummy2, true); got != "webhook_handler" {
		t.Fatalf("expected webhook_handler for unnamed original, got %s", got)
	}
}

// Test getWebhookProviderInfo additional branches
func TestGetWebhookProviderInfo_EdgeCases(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	if got := gen.getWebhookProviderInfo(&RouteInfo{WebhookHandler: badProv0{}}); got != nil {
		t.Fatalf("expected nil for no-return ProviderInfo, got %#v", got)
	}
	if got := gen.getWebhookProviderInfo(&RouteInfo{WebhookHandler: badProv1{}}); got != nil {
		t.Fatalf("expected nil for wrong-type ProviderInfo, got %#v", got)
	}
}

// Test extractEventTypesFromHandler and callEventTypesMethod edge cases
func TestEventTypesExtraction_EdgeCases(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	// No-return method => nil
	if types := gen.callEventTypesMethod(reflect.ValueOf(bad0{})); types != nil {
		t.Fatalf("expected nil from callEventTypesMethod with no results, got %#v", types)
	}
	// Wrong return type => nil via extractEventTypesFromHandler
	if types := gen.extractEventTypesFromHandler(badRet{}); types != nil {
		t.Fatalf("expected nil from extractEventTypesFromHandler with wrong return type, got %#v", types)
	}
}

// Test providerSpecificWebhookTypeRef when provider cannot be derived
func TestProviderSpecificWebhookTypeRef_NoProvider(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	// t with empty PkgPath -> providerFromPkgPath("") => ""
	typ := reflect.TypeOf(struct{}{})
	comps := &Components{Schemas: map[string]*Schema{}}
	s := &Schema{Type: "object"}
	if got := gen.providerSpecificWebhookTypeRef(typ, s, comps, "WebhookResponse"); got != s {
		t.Fatalf("expected unchanged schema when provider empty, got %#v", got)
	}
}

func TestProviderSpecificWebhookTypeRef_WithSchemaAndNilRegistry(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	// Use a type from foo provider so providerFromPkgPath(".../webhooks/foo") => foo
	typ := reflect.TypeOf(foo.WebhookResponse{})
	comps := &Components{} // Schemas is nil to cover init path
	s := &Schema{Type: "object"}
	got := gen.providerSpecificWebhookTypeRef(typ, s, comps, "WebhookResponse")
	if got == nil || got.Ref != "#/components/schemas/FooWebhookResponse" {
		t.Fatalf("expected FooWebhookResponse ref, got %#v", got)
	}
	if comps.Schemas == nil {
		t.Fatalf("expected Schemas map to be initialized")
	}
}

// resolveConcreteType additional branches
type rc struct{}

func (rc) Concrete() struct{ A int } { return struct{ A int }{42} }
func (rc) Interface() interface{}    { return 123 }

func TestResolveConcreteType_Additional(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	v := reflect.ValueOf(rc{})
	// returnType not interface{} -> early return
	m1 := v.MethodByName("Concrete")
	rt1 := gen.resolveConcreteType(m1, m1.Type(), m1.Type().Out(0), "Concrete")
	if rt1.Kind() != reflect.Struct {
		t.Fatalf("expected struct early return, got %v", rt1)
	}
	// interface{} return but unknown method name -> default branch
	m2 := v.MethodByName("Interface")
	rt2 := gen.resolveConcreteType(m2, m2.Type(), m2.Type().Out(0), "Unknown")
	if rt2.Kind() != reflect.Interface {
		t.Fatalf("expected interface type when name unknown, got %v", rt2)
	}
}

// Additional coverage for renameGenericWebhookRequestIfNeeded
func TestRenameGenericWebhookRequest_ComponentInitAndUnresolvedRef(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	reqType := reflect.TypeOf(foo.WebhookRequest{})
	comps := &Components{} // Schemas nil to trigger init branch
	// Provide a schema that references an absent component to hit unresolved-ref path
	s := &Schema{Ref: "#/components/schemas/WebhookRequest"}
	got := gen.renameGenericWebhookRequestIfNeeded(reqType, s, comps)
	if got == nil || got.Ref != "#/components/schemas/FooWebhookRequest" {
		t.Fatalf("expected FooWebhookRequest ref, got %#v", got)
	}
	if comps.Schemas == nil {
		t.Fatalf("expected Schemas map initialized")
	}
}

// Additional coverage for providerSpecificWebhookTypeRef unresolved ref and nil schema
func TestProviderSpecificWebhookTypeRef_UnresolvedRefAndNilSchema(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(nil, nil)
	typ := reflect.TypeOf(foo.WebhookResponse{})
	comps := &Components{Schemas: map[string]*Schema{}}
	// Unresolved ref -> falls back to using provided schema
	s := &Schema{Ref: "#/components/schemas/DoesNotExist"}
	got := gen.providerSpecificWebhookTypeRef(typ, s, comps, "WebhookResponse")
	if got == nil || got.Ref != "#/components/schemas/FooWebhookResponse" {
		t.Fatalf("expected FooWebhookResponse ref, got %#v", got)
	}
	// Nil schema -> return unchanged (nil)
	if got2 := gen.providerSpecificWebhookTypeRef(typ, nil, comps, "WebhookResponse"); got2 != nil {
		t.Fatalf("expected nil when schema is nil, got %#v", got2)
	}
}
