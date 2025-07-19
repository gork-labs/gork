package handlers

import (
	"context"

	"github.com/example/openapi-gen/pkg/unions"
)

type Option1 struct {
	// Option1Field is the first option field
	Option1Field string `json:"option1Field"`
}

type Option2 struct {
	// Option2Field is the second option field
	Option2Field string `json:"option2Field"`
}

// PaymentOptions represents possible payment methods
type PaymentOptions struct {
	Option1 *Option1
	Option2 *Option2
}

// AnyOfWithoutWrapperReq is a union type that accepts either Option1 or Option2
// The actual implementation is in oneof_gen.go (auto-generated)
type AnyOfWithoutWrapperReq unions.OneOf[PaymentOptions]

type BodyWithoutWrapperResp struct {
	Message string `json:"message"`
}

func BodyWithoutWrapperHandler(ctx context.Context, req *AnyOfWithoutWrapperReq) (*BodyWithoutWrapperResp, error) {
	if req.Option1 != nil {
		return &BodyWithoutWrapperResp{Message: "Handled Option1: " + req.Option1.Option1Field}, nil
	}

	if req.Option2 != nil {
		return &BodyWithoutWrapperResp{Message: "Handled Option2: " + req.Option2.Option2Field}, nil
	}

	return nil, nil
}

type AnyOfUnion2 struct {
	Option unions.Union2[Option1, Option2] `json:"option"`
}

type BodyWithWrapperResp struct {
	Message string `json:"message"`
}

func BodyWithWrapperHandler(ctx context.Context, req *AnyOfUnion2) (*BodyWithWrapperResp, error) {
	if req.Option.A != nil {
		return &BodyWithWrapperResp{Message: "Handled Option1: " + req.Option.A.Option1Field}, nil
	}

	if req.Option.B != nil {
		return &BodyWithWrapperResp{Message: "Handled Option2: " + req.Option.B.Option2Field}, nil
	}

	return nil, nil
}
