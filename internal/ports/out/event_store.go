package out

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

// EventStore is the append-only log every state change is captured in -
// see docs/adr/0019-event-sourcing-transactional-outbox.md.
type EventStore interface {
	Append(ctx context.Context, event domain.Event) (domain.Sequence, error)
}
