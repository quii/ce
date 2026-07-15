package inprocess_test

import (
	"testing"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/specifications"
)

func TestGreeting(t *testing.T) {
	greetings := memory.NewGreetingFinder()
	useCase := in.NewGetGreetingUseCase(greetings)

	specifications.GreetingSpecification(t, useCase)
}
