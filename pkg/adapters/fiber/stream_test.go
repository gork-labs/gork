package fiber

import (
	"context"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gork-labs/gork/pkg/api"
)

type streamTestEvents struct {
	Ping *struct{} `gork:"ping"`
}

func TestStreamRouteRejected(t *testing.T) {
	handler := func(context.Context, TestRequest, *api.Stream[streamTestEvents]) error { return nil }

	tests := []struct {
		name   string
		router func() *Router
		want   string
	}{
		{"root router", func() *Router { return NewRouter(fiber.New()) }, "fiber adapter does not support stream handlers: GET /live"},
		{"group router", func() *Router { return NewRouter(fiber.New()).Group("/v1") }, "fiber adapter does not support stream handlers: GET /v1/live"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := tt.router()
			defer func() {
				if r := recover(); r != tt.want {
					t.Errorf("expected panic %q, got %v", tt.want, r)
				}
			}()
			router.Get("/live", handler)
		})
	}
}
