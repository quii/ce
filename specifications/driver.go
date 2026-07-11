package specifications

import "context"

type Driver interface {
	Greeting(ctx context.Context) (string, error)
}
