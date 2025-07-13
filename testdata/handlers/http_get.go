package handlers

import "context"

type GetWithoutQueryParamsReq struct {
}

type GetWithoutQueryParamsResp struct {
	// Message is a simple message
	Message string `json:"message"`
}

func GetWithoutQueryParams(ctx context.Context, req *GetWithoutQueryParamsReq) (*GetWithoutQueryParamsResp, error) {
	// Handle logic here
	return &GetWithoutQueryParamsResp{Message: "Hello, world!"}, nil
}

type GetWithQueryParamsReq struct {
	// Name is the name to greet
	Name string `goapi:"my-name,in=query"`
}

type GetWithQueryParamsResp struct {
	// Message is a personalized greeting message
	Message string `json:"message"`
}

func GetWithQueryParams(ctx context.Context, req *GetWithQueryParamsReq) (*GetWithQueryParamsResp, error) {
	// Handle logic here
	return &GetWithQueryParamsResp{Message: "Hello, " + req.Name + "!"}, nil
}
