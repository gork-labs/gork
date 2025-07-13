package handlers

import "context"

type PostWithJsonBodyReq struct {
	// Name is the name to greet
	Name string `json:"name"`
}

type PostWithJsonBodyResp struct {
	// Message is a personalized greeting message
	Message string `json:"message"`
}

func PostWithJsonBody(ctx context.Context, req *PostWithJsonBodyReq) (*PostWithJsonBodyResp, error) {
	// Handle logic here
	return &PostWithJsonBodyResp{Message: "Hello, " + req.Name + "!"}, nil
}

type PostWithJsonBodyAndQueryParamsReq struct {
	// Name is the name to greet
	Name string `goapi:"my-name,in=query"`
	// Age is the age of the person to greet
	Age int `json:"age"`
}

type PostWithJsonBodyAndQueryParamsResp struct {
	// Message is a personalized greeting message
	Message string `json:"message"`
}

func PostWithJsonBodyAndQueryParams(ctx context.Context, req *PostWithJsonBodyAndQueryParamsReq) (*PostWithJsonBodyAndQueryParamsResp, error) {
	return &PostWithJsonBodyAndQueryParamsResp{Message: "Hello, " + req.Name + "!"}, nil
}
