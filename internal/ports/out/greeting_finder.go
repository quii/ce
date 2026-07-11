package out

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

type GreetingFinder interface {
	FindGreeting(ctx context.Context) (domain.Greeting, error)
}
