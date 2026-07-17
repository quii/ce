// client.gen.go is generated from api/openapi.yaml - run `go generate ./...`
// after editing the spec.
//
//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
package container

import (
	"context"
	"fmt"
	"net/http"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type Driver struct {
	client *ClientWithResponses
}

func New(baseURL string) *Driver {
	client, err := NewClientWithResponses(baseURL)
	if err != nil {
		panic(err)
	}
	return &Driver{client: client}
}

func (d *Driver) Greet(ctx context.Context, cmd in.GetGreetingCommand) (domain.Greeting, error) {
	var params GetGreetingParams
	if cmd.Name != "" {
		params.Name = &cmd.Name
	}

	resp, err := d.client.GetGreetingWithResponse(ctx, &params)
	if err != nil {
		return "", err
	}

	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode())
	}

	return domain.Greeting(resp.JSON200.Greeting), nil
}
