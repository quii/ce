package specifications

import "context"

type Driver interface {
	Greeting(ctx context.Context, name string) (string, error)
}
