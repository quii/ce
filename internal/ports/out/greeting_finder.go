package out

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

type GreetingFinder interface {
	FindPrefix(ctx context.Context) (domain.Prefix, error)
}
