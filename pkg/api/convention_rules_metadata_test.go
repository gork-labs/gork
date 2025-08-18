package api

import (
	"reflect"
	"testing"
)

// Request type with nested structs and rule tags to exercise rules metadata extraction
type rulesRequest struct {
	Path struct {
		// multiple invocations, quotes with commas, numbers, booleans, context var
		A string `rule:"allow('a,b'), deny(1), flag(true), zero(false), ctx($user), call(nested(1, 'x,y'), 2)"`
	}
	Body struct {
		B string `rule:"group(and(foo(), or(bar(), not baz())))"`
	}
}

func TestAddRulesExtension_And_EntriesParsing(t *testing.T) {
	spec := &OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}
	gen := NewConventionOpenAPIGenerator(spec, NewDocExtractor())

	// Prime extractor docs for one rule to test description inclusion in buildRuleEntry
	gen.extractor.docs["allow"] = Documentation{Description: "Allow rule doc"}

	route := &RouteInfo{HandlerName: "RulesHandler", RequestType: reflect.TypeOf(rulesRequest{}), ResponseType: nil}
	op := &Operation{Responses: map[string]*Response{}}

	// Invoke addRulesExtension which walks struct, parses rule tags, and builds entries
	gen.addRulesExtension(route, op)

	if op.Extensions == nil {
		t.Fatalf("expected extensions to be populated")
	}
	x, ok := op.Extensions["x-rules"].([]map[string]interface{})
	if !ok || len(x) == 0 {
		t.Fatalf("expected x-rules entries, got %#v", op.Extensions["x-rules"])
	}

	// Ensure at least our documented entry is present with description and empty names are skipped
	foundAllow := false
	for _, e := range x {
		if e["name"] == "allow" {
			foundAllow = true
			if e["description"] != "Allow rule doc" {
				t.Fatalf("allow description missing: %#v", e)
			}
		}
	}
	if !foundAllow {
		t.Fatalf("allow entry not found in x-rules: %#v", x)
	}
	// verify no empty-name entries
	for _, e := range x {
		if name, _ := e["name"].(string); name == "" {
			t.Fatalf("unexpected empty name entry in x-rules: %#v", x)
		}
	}
}

func TestParseRuleInvocations_And_SplitTopLevelLocal(t *testing.T) {
	// complex string ensures commas inside quotes and nested parentheses are respected
	s := "allow('a,b'),deny(1),flag(true),zero(false),ctx($user),call(nested(1,'x,y'),2),group(and(foo(),or(bar(),not baz())))"
	invs, err := parseRuleInvocations(s)
	if err != nil {
		t.Fatalf("parseRuleInvocations err: %v", err)
	}
	got := make([]string, 0, len(invs))
	for _, inv := range invs {
		got = append(got, inv.Name)
	}
	want := []string{"allow", "deny", "flag", "zero", "ctx", "call", "group"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names mismatch at %d: %v vs %v", i, got[i], want[i])
		}
	}

	// error path: unmatched parenthesis
	if _, err := splitTopLevelLocal("foo(bar", ','); err == nil {
		t.Fatalf("expected error for unmatched parenthesis")
	}
}
