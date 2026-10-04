package api

import (
	"context"
	"reflect"
	"testing"
)

type recursiveNode struct {
	Name     string          `gork:"name"`
	Children []recursiveNode `gork:"children"`
	Parent   *recursiveNode  `gork:"parent"`
}

type recursiveTree struct {
	Left  *recursiveTree `gork:"left"`
	Right *recursiveTree `gork:"right"`
}

type recursiveNodeResponse struct {
	Body recursiveNode
}

type recursiveTreeRequest struct {
	Body struct {
		Tree recursiveTree `gork:"tree"`
	}
}

func TestRecursiveTypesGiveComponentRefs(t *testing.T) {
	registry := NewRouteRegistry()
	router := NewTypedRouter[*struct{}](nil, registry, "", nil, &HTTPParameterAdapter{}, nil)
	router.Get("/nodes", func(context.Context, struct{}) (*recursiveNodeResponse, error) { return nil, nil })
	router.Post("/trees", func(context.Context, recursiveTreeRequest) error { return nil })

	schemas := GenerateOpenAPI(registry).Components.Schemas

	nodeRef := "#/components/schemas/recursiveNode"
	node := schemas["recursiveNode"]
	if got := node.Properties["children"].Items.Ref; got != nodeRef {
		t.Errorf("children items ref = %q, want %q", got, nodeRef)
	}
	if got := node.Properties["parent"].AnyOf[0].Ref; got != nodeRef {
		t.Errorf("parent ref = %q, want %q", got, nodeRef)
	}
	if want := []string{"name", "children", "parent"}; !reflect.DeepEqual(node.Required, want) {
		t.Errorf("required = %v, want %v", node.Required, want)
	}

	treeRef := "#/components/schemas/recursiveTree"
	tree := schemas["recursiveTree"]
	if tree.Type != "object" || tree.Properties["left"].AnyOf[0].Ref != treeRef || tree.Properties["right"].AnyOf[0].Ref != treeRef {
		t.Errorf("recursiveTree = %+v, want an object with left and right refs to itself", tree)
	}
	if got := schemas["recursiveTreeBody"].Properties["tree"].Ref; got != treeRef {
		t.Errorf("request tree ref = %q, want %q", got, treeRef)
	}
}
