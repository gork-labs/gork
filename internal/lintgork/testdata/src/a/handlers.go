package a

import (
	"context"

	"github.com/gork-labs/gork/pkg/api"
)

func handleInvalid(ctx context.Context, req InvalidRequest) (struct{}, error) {
	return struct{}{}, nil
}

func handleNonStructSection(ctx context.Context, req NonStructSectionRequest) (struct{}, error) {
	return struct{}{}, nil
}

func handleNamedBody(ctx context.Context, req NamedBodyRequest) (struct{}, error) {
	return struct{}{}, nil
}

func handleNamedSection(ctx context.Context, req NamedSectionRequest) (struct{}, error) {
	return struct{}{}, nil
}

func handleExtraFields(ctx context.Context, req ExtraFieldsRequest) (struct{}, error) {
	return struct{}{}, nil
}

func handleBadUpload(ctx context.Context, req BadUploadRequest) (struct{}, error) {
	return struct{}{}, nil
}

type server struct{}

func (server) handlePointerSection(ctx context.Context, req *PointerSectionRequest) (struct{}, error) {
	return struct{}{}, nil
}

func handleAnonymous(ctx context.Context, req struct{ Query int }) (struct{}, error) {
	return struct{}{}, nil
}

func handleForeign(ctx context.Context, req api.File) (struct{}, error) {
	return struct{}{}, nil
}

func notHandler(ctx context.Context, req InvalidRequest) error {
	return nil
}
