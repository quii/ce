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
	name := domain.NewName(cmd.Name)
	if name.IsBlank() {
		return uc.greetings.FindGreeting(ctx)
	}
	return name.Greet(), nil
}
