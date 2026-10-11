package requeststream

import (
	"context"

	"github.com/gork-labs/gork/pkg/api"
)

type Event struct {
	Value *string `gork:"value"`
}

type Request struct {
	Query string // want "section 'Query' must be a struct type"
}

func Handle(ctx context.Context, req Request, stream *api.Stream[Event]) error {
	return nil
}
