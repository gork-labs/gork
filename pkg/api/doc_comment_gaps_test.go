package api

import (
	"context"
	"net/http"
	"testing"
)

func gapListItems(context.Context, struct{}) error { return nil }

type gapHandlers struct{}

func (*gapHandlers) List(context.Context, struct{}) error { return nil }

type gapRow struct {
	// Name is the name of the row
	Name string `gork:"name"`
}

type gapGrouped struct {
	// Size is the size of the group
	Size int `gork:"size"`
}

type gapRowsResponse struct {
	Body struct {
		Row     gapRow     `gork:"row"`
		Grouped gapGrouped `gork:"grouped"`
	}
}

type gapRedirectResponse struct {
	Headers struct {
		// Location is the page that the browser opens next
		Location string `gork:"Location"`
	}
}

type gapListQuery struct {
	// Limit is the maximum number of items
	Limit int `gork:"limit"`
}

type gapListRequest struct {
	Query gapListQuery
}

const gapSource = `package api

// gapListItems lists the items.
func gapListItems() {}

type gapHandlers struct{}

// List lists the items of gapHandlers.
func (*gapHandlers) List() {}

type gapOther struct{}

// List is the method of another type.
func (gapOther) List() {}

type gapRow struct {
	// Name is the name of the row
	Name string ` + "`gork:\"name\"`" + `
}

type (
	// gapGrouped is a type in a group.
	gapGrouped struct {
		// Size is the size of the group
		Size int ` + "`gork:\"size\"`" + `
	}
)

type gapRedirectResponse struct {
	Headers struct {
		// Location is the page that the browser opens next
		Location string ` + "`gork:\"Location\"`" + `
	}
}

type gapListQuery struct {
	// Limit is the maximum number of items
	Limit int ` + "`gork:\"limit\"`" + `
}

type gapListRequest struct {
	Query gapListQuery
}
`

const gapOtherSource = `package other

// gapListItems is a function of another package.
func gapListItems() {}
`

func gapSpec(t *testing.T) *OpenAPISpec {
	t.Helper()
	extractor := parseFixtures(t, map[string]string{
		"pkg/api/fixture.go":   gapSource,
		"pkg/other/fixture.go": gapOtherSource,
	})
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/items", gapListItems)
	router.Get("/handlers/items", (&gapHandlers{}).List)
	router.Get("/rows", func(context.Context, struct{}) (*gapRowsResponse, error) { return nil, nil })
	router.Get("/redirect", func(context.Context, struct{}) (*gapRedirectResponse, error) { return nil, nil }, WithStatus(http.StatusSeeOther))
	router.Get("/list", func(context.Context, gapListRequest) error { return nil })
	return GenerateOpenAPIWithDocs(registry, extractor)
}

func TestOperationDescriptionIsTheDocOfTheHandler(t *testing.T) {
	spec := gapSpec(t)
	if got := spec.Paths["/items"].Get.Description; got != "gapListItems lists the items." {
		t.Errorf("function handler description = %q", got)
	}
	if got := spec.Paths["/handlers/items"].Get.Description; got != "List lists the items of gapHandlers." {
		t.Errorf("method handler description = %q", got)
	}
}

func TestFieldDocsOfTypeWithoutDocComment(t *testing.T) {
	schemas := gapSpec(t).Components.Schemas
	if got := schemas["gapRow"].Properties["name"].Description; got != "Name is the name of the row" {
		t.Errorf("gapRow.name description = %q", got)
	}
	if got := schemas["gapGrouped"].Description; got != "gapGrouped is a type in a group." {
		t.Errorf("gapGrouped description = %q", got)
	}
	if got := schemas["gapGrouped"].Properties["size"].Description; got != "Size is the size of the group" {
		t.Errorf("gapGrouped.size description = %q", got)
	}
}

func TestResponseHeaderDescriptionIsTheFieldDoc(t *testing.T) {
	header := gapSpec(t).Paths["/redirect"].Get.Responses["303"].Headers["Location"]
	if got := header.Description; got != "Location is the page that the browser opens next" {
		t.Errorf("Location description = %q", got)
	}
}

func TestParameterDocOfNamedSectionType(t *testing.T) {
	params := gapSpec(t).Paths["/list"].Get.Parameters
	if len(params) != 1 || params[0].Description != "Limit is the maximum number of items" {
		t.Errorf("parameters = %+v, want limit with its field doc", params)
	}
}
