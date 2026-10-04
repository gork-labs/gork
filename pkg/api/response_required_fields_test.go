package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/gork-labs/gork/pkg/unions"
)

type requiredAddress struct {
	City string `gork:"city"`
	Zip  string `gork:"zip" validate:"required"`
}

type requiredBase struct {
	Base string `gork:"base"`
}

type requiredItem struct {
	requiredBase
	Name     string  `gork:"name"`
	Note     *string `gork:"note"`
	JSONName int     `json:"json_name,omitempty"`
	Untagged string
	Skipped  string          `gork:"-"`
	Address  requiredAddress `gork:"address"`
	Parent   *struct {
		ID string `gork:"id"`
	} `gork:"parent"`
	Children []requiredChild             `gork:"children"`
	Extra    map[string]requiredMapValue `gork:"extra"`
}

type requiredMapValue struct {
	Name string `json:"name"`
}

type requiredChild struct {
	ID string `gork:"id"`
}

type requiredFilter struct {
	Tag   string `gork:"tag"`
	Limit int    `gork:"limit" validate:"required"`
}

type requiredItemRequest struct {
	Body struct {
		Title  string         `gork:"title"`
		Filter requiredFilter `gork:"filter" validate:"required"`
	}
}

type requiredItemResponse struct {
	Body requiredItem
}

type requiredAnonResponse struct {
	Body struct {
		X string `gork:"x"`
	}
}

type requiredCard struct {
	Type   string `gork:"type,discriminator=card"`
	Number string `gork:"number"`
}

type requiredBank struct {
	Type string `gork:"type,discriminator=bank"`
	IBAN string `gork:"iban"`
}

type requiredMethodResponse struct {
	Body unions.Union2[requiredCard, requiredBank]
}

type requiredEvent struct {
	A string `gork:"a"`
	B *int   `gork:"b"`
}

type requiredEvents struct {
	Item *requiredEvent `gork:"item"`
}

func TestResponseFieldsRequiredWhenWritten(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Post("/items", func(context.Context, requiredItemRequest) (*requiredItemResponse, error) { return nil, nil })
	router.Get("/anon", func(context.Context, struct{}) (*requiredAnonResponse, error) { return nil, nil })
	router.Get("/method", func(context.Context, struct{}) (*requiredMethodResponse, error) { return nil, nil })
	router.Get("/events", func(context.Context, struct{}, *Stream[requiredEvents]) error { return nil })

	schemas := GenerateOpenAPI(registry).Components.Schemas

	item := schemas["requiredItem"]
	if want := []string{"name", "note", "json_name", "address", "parent", "children", "extra"}; !reflect.DeepEqual(item.Required, want) {
		t.Errorf("requiredItem required = %v, want %v", item.Required, want)
	}
	if parent := item.Properties["parent"].AnyOf[0]; !reflect.DeepEqual(parent.Required, []string{"id"}) {
		t.Errorf("parent required = %v, want [id]", parent.Required)
	}
	if got := schemas["requiredChild"].Required; !reflect.DeepEqual(got, []string{"id"}) {
		t.Errorf("requiredChild required = %v, want [id]", got)
	}
	if got := schemas["requiredMapValue"].Required; got != nil {
		t.Errorf("requiredMapValue required = %v, want none because encoding/json writes map values", got)
	}
	if got := schemas["requiredAddress"].Required; !reflect.DeepEqual(got, []string{"zip", "city"}) {
		t.Errorf("requiredAddress required = %v, want [zip city]", got)
	}
	if got := schemas["requiredAnonResponse"].Required; !reflect.DeepEqual(got, []string{"x"}) {
		t.Errorf("requiredAnonResponse required = %v, want [x]", got)
	}
	if got := schemas["requiredCard"].Required; !reflect.DeepEqual(got, []string{"type", "number"}) {
		t.Errorf("requiredCard required = %v, want [type number]", got)
	}
	if got := schemas["requiredEvent"].Required; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("requiredEvent required = %v, want [a b]", got)
	}

	if got := schemas["requiredItemBody"].Required; !reflect.DeepEqual(got, []string{"filter"}) {
		t.Errorf("request body required = %v, want [filter]", got)
	}
	if got := schemas["requiredFilter"].Required; !reflect.DeepEqual(got, []string{"limit"}) {
		t.Errorf("requiredFilter required = %v, want [limit]", got)
	}
}
