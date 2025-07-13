package testdata

import (
	"net/http"
	"github.com/example/openapi-gen/pkg/api"
	"github.com/example/openapi-gen/testdata/handlers"
)

func MakeRoutes() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /basic/hello", api.HandlerFunc(handlers.GetWithoutQueryParams, api.WithTags("basic")))
	mux.HandleFunc("GET /basic/hello-with-query", api.HandlerFunc(handlers.GetWithQueryParams, api.WithTags("basic")))

	mux.HandleFunc("POST /basic/hello-with-json", api.HandlerFunc(handlers.PostWithJsonBody, api.WithTags("basic")))
	mux.HandleFunc("POST /basic/hello-with-json-and-query", api.HandlerFunc(handlers.PostWithJsonBodyAndQueryParams, api.WithTags("basic")))

	mux.HandleFunc("POST /unions/any-of-without-wrapper", api.HandlerFunc(handlers.BodyWithoutWrapperHandler, api.WithTags("unions")))
	mux.HandleFunc("POST /unions/any-of-union2", api.HandlerFunc(handlers.BodyWithWrapperHandler, api.WithTags("unions")))
}
