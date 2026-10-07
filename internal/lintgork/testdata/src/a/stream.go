package a

import (
	"context"

	"github.com/gork-labs/gork/pkg/api"
)

type FeedRow struct {
	ID string `gork:"id"`
}

type LiveEvents struct {
	Feed  *FeedRow  `gork:"feed"`
	Reset *struct{} `gork:"reset"`
}

type BadEvents struct {
	Value    FeedRow  `gork:"value"`  // want "stream event field 'Value' must be a pointer"
	Untagged *FeedRow // want "stream event field 'Untagged' missing gork tag"
	hidden   *FeedRow `gork:"hidden"` // want "stream event field 'hidden' must be exported"
}

type LiveRequest struct {
	Query struct {
		Topic string `gork:"topic"`
	}
}

func Live(ctx context.Context, req LiveRequest, stream *api.Stream[LiveEvents]) error {
	return nil
}

func BadLive(ctx context.Context, req LiveRequest, stream *api.Stream[BadEvents]) error {
	return nil
}

func ScalarLive(ctx context.Context, req LiveRequest, stream *api.Stream[string]) error {
	return nil
}

func ValueParam(ctx context.Context, req LiveRequest, extra FeedRow) error {
	return nil
}

func OtherPointer(ctx context.Context, req LiveRequest, extra *FeedRow) error {
	return nil
}

func BasicPointer(ctx context.Context, req LiveRequest, extra *int) error {
	return nil
}

func UniversePointer(ctx context.Context, req LiveRequest, extra *error) error {
	return nil
}

type ImageResponse struct {
	Headers struct {
		CacheControl string `gork:"Cache-Control"`
	}
	Body api.Binary
}
