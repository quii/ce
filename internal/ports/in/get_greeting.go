package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type GetGreetingCommand struct{}

type GetGreetingUseCase struct {
	greetings out.GreetingFinder
}

func NewGetGreetingUseCase(greetings out.GreetingFinder) *GetGreetingUseCase {
	return &GetGreetingUseCase{greetings: greetings}
}

func (uc *GetGreetingUseCase) Handle(ctx context.Context, _ GetGreetingCommand) (domain.Greeting, error) {
	return uc.greetings.FindGreeting(ctx)
}
