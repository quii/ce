package memory

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

type GreetingFinder struct{}

func NewGreetingFinder() *GreetingFinder {
	return &GreetingFinder{}
}

func (f *GreetingFinder) FindGreeting(_ context.Context) (domain.Greeting, error) {
	return domain.Greeting("Hello, World!"), nil
}
