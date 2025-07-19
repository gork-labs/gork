// Code generated test file for adapter
package api

import (
	"context"
	"net/http/httptest"
	"testing"
)

type queryTestReq struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Count   uint     `json:"count"`
	Active  bool     `json:"active"`
	Tags    []string `json:"tags"`
	CSVTags []string `json:"csvTags"`
}

func TestParseQueryParams(t *testing.T) {
	reqStruct := &queryTestReq{}

	r := httptest.NewRequest("GET", "/test?name=Alice&age=30&count=5&active=true&tags=foo&tags=bar&csvTags=alpha,beta", nil)

	parseQueryParams(r, reqStruct)

	if reqStruct.Name != "Alice" {
		t.Errorf("expected Name to be 'Alice', got %q", reqStruct.Name)
	}
	if reqStruct.Age != 30 {
		t.Errorf("expected Age to be 30, got %d", reqStruct.Age)
	}
	if reqStruct.Count != 5 {
		t.Errorf("expected Count to be 5, got %d", reqStruct.Count)
	}
	if !reqStruct.Active {
		t.Errorf("expected Active to be true")
	}
	expectedTags := []string{"foo", "bar"}
	if len(reqStruct.Tags) != len(expectedTags) {
		t.Fatalf("expected %d tags, got %d", len(expectedTags), len(reqStruct.Tags))
	}
	for i, tag := range expectedTags {
		if reqStruct.Tags[i] != tag {
			t.Errorf("expected Tags[%d] to be %q, got %q", i, tag, reqStruct.Tags[i])
		}
	}

	// Verify CSV-style slice parsing with a single additional check
	if len(reqStruct.CSVTags) != 2 || reqStruct.CSVTags[0] != "alpha" || reqStruct.CSVTags[1] != "beta" {
		t.Errorf("expected CSVTags to be [alpha beta], got %v", reqStruct.CSVTags)
	}
}

// DummyHandler is used to test getFunctionName
func DummyHandler(ctx context.Context, req queryTestReq) (string, error) {
	return "", nil
}

func TestGetFunctionName(t *testing.T) {
	name := getFunctionName(DummyHandler)
	if name != "DummyHandler" {
		t.Errorf("expected function name 'DummyHandler', got %q", name)
	}
}
