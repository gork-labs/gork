package requesthandler

import "context"

type Request struct {
	Method string
	Path   string // want "section 'Path' must be a struct type"
	Query  string // want "section 'Query' must be a struct type"
	Status int
}

func Handle(ctx context.Context, req Request) (struct{}, error) {
	return struct{}{}, nil
}
