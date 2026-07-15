package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type GetGreetingCommand struct {
	Name domain.Name
}

type GetGreetingUseCase struct {
	greetings out.GreetingFinder
}

func NewGetGreetingUseCase(greetings out.GreetingFinder) *GetGreetingUseCase {
	return &GetGreetingUseCase{greetings: greetings}
}

func (uc *GetGreetingUseCase) Handle(ctx context.Context, cmd GetGreetingCommand) (domain.Greeting, error) {
	if cmd.Name.IsBlank() {
		return uc.greetings.FindGreeting(ctx)
	}
	return cmd.Name.Greet(), nil
}
