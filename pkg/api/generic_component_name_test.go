package api

import (
	"context"
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"
)

type envelope[T any] struct {
	Data T `gork:"data"`
}

type item struct {
	Name string `gork:"name"`
}

type user struct {
	ID string `gork:"id"`
}

type page[T any] struct {
	Items []T `gork:"items"`
}

type pair[A, B any] struct {
	First  A `gork:"first"`
	Second B `gork:"second"`
}

type Point struct {
	X int `gork:"x"`
}

func TestGenericComponentNames(t *testing.T) {
	tests := []struct {
		typ  reflect.Type
		want string
	}{
		{reflect.TypeOf(envelope[item]{}), "envelope_item"},
		{reflect.TypeOf(envelope[[]item]{}), "envelope_Array_item"},
		{reflect.TypeOf(envelope[[3]item]{}), "envelope_Array3_item"},
		{reflect.TypeOf(envelope[*item]{}), "envelope_Nullable_item"},
		{reflect.TypeOf(envelope[string]{}), "envelope_string"},
		{reflect.TypeOf(envelope[map[string]item]{}), "envelope_Map_string_item"},
		{reflect.TypeOf(envelope[map[pair[item, string]][]*item]{}), "envelope_Map_pair_item_string_Array_Nullable_item"},
		{reflect.TypeOf(envelope[page[item]]{}), "envelope_page_item"},
		{reflect.TypeOf(envelope[envelope[[]item]]{}), "envelope_envelope_Array_item"},
		{reflect.TypeOf(pair[item, user]{}), "pair_item_user"},
		{reflect.TypeOf(pair[item, image.Point]{}), "pair_item_Point"},
		{reflect.TypeOf(pair[func(int, string), item]{}), "pair_func_int__string__item"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := sanitizeSchemaName(tt.typ.Name()); got != tt.want {
				t.Errorf("sanitizeSchemaName(%q) = %q, want %q", tt.typ.Name(), got, tt.want)
			}
		})
	}
}

type itemResponse struct {
	Body envelope[item]
}

type itemListResponse struct {
	Body envelope[[]item]
}

type itemMapResponse struct {
	Body envelope[map[string]item]
}

type itemPageResponse struct {
	Body envelope[page[item]]
}

type itemUserResponse struct {
	Body pair[item, user]
}

func TestGenericInstancesGetDistinctComponents(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/item", func(context.Context, struct{}) (*itemResponse, error) { return nil, nil })
	router.Get("/items", func(context.Context, struct{}) (*itemListResponse, error) { return nil, nil })
	router.Get("/map", func(context.Context, struct{}) (*itemMapResponse, error) { return nil, nil })
	router.Get("/page", func(context.Context, struct{}) (*itemPageResponse, error) { return nil, nil })
	router.Get("/pair", func(context.Context, struct{}) (*itemUserResponse, error) { return nil, nil })

	spec := GenerateOpenAPI(registry)
	schemas := spec.Components.Schemas
	itemRef := "#/components/schemas/item"

	for path, want := range map[string]string{
		"/item":  "envelope_item",
		"/items": "envelope_Array_item",
		"/map":   "envelope_Map_string_item",
		"/page":  "envelope_page_item",
		"/pair":  "pair_item_user",
	} {
		if got := spec.Paths[path].Get.Responses["200"].Content["application/json"].Schema.Ref; got != "#/components/schemas/"+want {
			t.Errorf("%s response ref = %q, want %q", path, got, want)
		}
	}

	if got := schemas["envelope_item"].Properties["data"].Ref; got != itemRef {
		t.Errorf("envelope_item data ref = %q, want %q", got, itemRef)
	}
	if data := schemas["envelope_Array_item"].Properties["data"]; data.Type != "array" || data.Items.Ref != itemRef {
		t.Errorf("envelope_Array_item data = %+v, want an array of %s", data, itemRef)
	}
	if got := schemas["envelope_Map_string_item"].Properties["data"].AdditionalProperties.Ref; got != itemRef {
		t.Errorf("envelope_Map_string_item data value ref = %q, want %q", got, itemRef)
	}
	if got := schemas["envelope_page_item"].Properties["data"].Ref; got != "#/components/schemas/page_item" {
		t.Errorf("envelope_page_item data ref = %q, want page_item", got)
	}
	if got := schemas["page_item"].Properties["items"].Items.Ref; got != itemRef {
		t.Errorf("page_item items ref = %q, want %q", got, itemRef)
	}
	if got := schemas["pair_item_user"].Properties["second"].Ref; got != "#/components/schemas/user" {
		t.Errorf("pair_item_user second ref = %q, want user", got)
	}
}

type localPointResponse struct {
	Body envelope[Point]
}

type imagePointResponse struct {
	Body envelope[image.Point]
}

func TestComponentNameCollisionPanics(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/local", func(context.Context, struct{}) (*localPointResponse, error) { return nil, nil })
	router.Get("/image", func(context.Context, struct{}) (*imagePointResponse, error) { return nil, nil })

	defer func() {
		msg := fmt.Sprint(recover())
		for _, want := range []string{
			"github.com/gork-labs/gork/pkg/api.envelope[github.com/gork-labs/gork/pkg/api.Point]",
			"github.com/gork-labs/gork/pkg/api.envelope[image.Point]",
			`"envelope_Point"`,
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("panic = %q, want it to contain %q", msg, want)
			}
		}
	}()
	GenerateOpenAPI(registry)
}
