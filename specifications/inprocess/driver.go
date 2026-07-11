package inprocess

import (
	"context"

	"github.com/quii/ce/internal/ports/in"
)

type Driver struct {
	useCase *in.GetGreetingUseCase
}

func New(useCase *in.GetGreetingUseCase) *Driver {
	return &Driver{useCase: useCase}
}

func (d *Driver) Greeting(ctx context.Context) (string, error) {
	greeting, err := d.useCase.Handle(ctx, in.GetGreetingCommand{})
	if err != nil {
		return "", err
	}
	return string(greeting), nil
}
