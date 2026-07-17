package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type GetGreetingCommand struct {
	Name string
}

type Greeter interface {
	Greet(ctx context.Context, cmd GetGreetingCommand) (domain.Greeting, error)
}

type getGreetingUseCase struct {
	greetings out.GreetingFinder
}

func NewGetGreetingUseCase(greetings out.GreetingFinder) Greeter {
	return &getGreetingUseCase{greetings: greetings}
}

func (uc *getGreetingUseCase) Greet(ctx context.Context, cmd GetGreetingCommand) (domain.Greeting, error) {
	prefix, err := uc.greetings.FindPrefix(ctx)
	if err != nil {
		return "", err
	}
	return domain.NewName(cmd.Name).Greet(prefix), nil
}
