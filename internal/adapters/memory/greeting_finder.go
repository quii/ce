package memory

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

type GreetingFinder struct{}

func NewGreetingFinder() *GreetingFinder {
	return &GreetingFinder{}
}

func (f *GreetingFinder) FindPrefix(_ context.Context) (domain.Prefix, error) {
	return domain.Prefix("Hello"), nil
}
