// Package httpapi is the HTTP adapter. server.gen.go is generated from
// api/openapi.yaml - run `go generate ./...` after editing the spec.
//
//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../api/openapi.yaml
package httpapi

import (
	"context"

	"github.com/quii/ce/internal/ports/in"
)

type GreetingHandler struct {
	useCase in.Greeter
}

func NewGreetingHandler(useCase in.Greeter) *GreetingHandler {
	return &GreetingHandler{useCase: useCase}
}

func (h *GreetingHandler) GetGreeting(ctx context.Context, request GetGreetingRequestObject) (GetGreetingResponseObject, error) {
	var name string
	if request.Params.Name != nil {
		name = *request.Params.Name
	}

	greeting, err := h.useCase.Greet(ctx, in.GetGreetingCommand{Name: name})
	if err != nil {
		return nil, err
	}

	return GetGreeting200JSONResponse{Greeting: string(greeting)}, nil
}
