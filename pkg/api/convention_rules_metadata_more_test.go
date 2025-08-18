package api

import (
	"reflect"
	"testing"
)

func TestAddRulesExtension_EarlyReturns(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, nil)
	// nil route
	op := &Operation{}
	gen.addRulesExtension(nil, op)
	if op.Extensions != nil {
		t.Fatalf("expected no extensions when route is nil")
	}
	// non-struct request type
	gen = NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, NewDocExtractor())
	op = &Operation{}
	gen.addRulesExtension(&RouteInfo{RequestType: reflect.TypeOf(42)}, op)
	if op.Extensions != nil {
		t.Fatalf("expected no extensions for non-struct req type")
	}
	// struct with no rule tags -> no x-rules
	type noRules struct{ Path struct{ X string } }
	op = &Operation{}
	gen.addRulesExtension(&RouteInfo{RequestType: reflect.TypeOf(noRules{})}, op)
	if op.Extensions != nil {
		t.Fatalf("expected no extensions when no rules found")
	}
}

func TestEntriesFromRuleTag_ErrorAndEmpty(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, NewDocExtractor())
	// malformed -> parse error -> nil
	if e := gen.entriesFromRuleTag("foo(bar"); e != nil {
		t.Fatalf("expected nil on parse error, got %#v", e)
	}
	// empty input -> nil
	if e := gen.entriesFromRuleTag(""); e != nil {
		t.Fatalf("expected nil for empty tag, got %#v", e)
	}
	// only separators / whitespace -> nil
	if e := gen.entriesFromRuleTag(" ,  , "); e != nil {
		t.Fatalf("expected nil for only separators, got %#v", e)
	}
	// blank item filtered
	entries := gen.entriesFromRuleTag(", foo() , ,  ")
	if len(entries) != 1 || entries[0]["name"] != "foo" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestEntriesFromRuleTag_EmptyNameSkipped(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, NewDocExtractor())
	// A part starting with '(' yields empty name before '(', which must be skipped, while the second is valid
	entries := gen.entriesFromRuleTag("(allow()), allow()")
	if len(entries) != 1 || entries[0]["name"] != "allow" {
		t.Fatalf("expected single allow entry, got %#v", entries)
	}
}

func TestBuildRuleEntry_Branches(t *testing.T) {
	// extractor nil -> only name
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, nil)
	e := gen.buildRuleEntry("foo")
	if e["name"] != "foo" || e["description"] != nil {
		t.Fatalf("unexpected entry without extractor: %#v", e)
	}
	// empty name -> still returns name only
	e = gen.buildRuleEntry("")
	if e["name"] != "" {
		t.Fatalf("expected empty name entry: %#v", e)
	}
}

func TestSplitTopLevelLocal_DoubleQuotes_And_UnmatchedClose(t *testing.T) {
	parts, err := splitTopLevelLocal(`foo("x,y"),bar("z")`, ',')
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(parts) != 2 || parts[0][:3] != "foo" || parts[1][:3] != "bar" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
	// unmatched closing
	if _, err := splitTopLevelLocal(")", ','); err == nil {
		t.Fatalf("expected error for unmatched close paren")
	}
}

func TestParseRuleInvocations_Empty(t *testing.T) {
	invs, err := parseRuleInvocations("  ")
	if err != nil || invs != nil {
		t.Fatalf("expected nil invocations for empty input: %v %#v", err, invs)
	}
}

func TestCollectRuleEntriesFromType_NonStructEarlyReturn(t *testing.T) {
	gen := NewConventionOpenAPIGenerator(&OpenAPISpec{Components: &Components{Schemas: map[string]*Schema{}}}, NewDocExtractor())
	if out := gen.collectRuleEntriesFromType(reflect.TypeOf(42)); out != nil {
		t.Fatalf("expected nil for non-struct type, got %#v", out)
	}
}
