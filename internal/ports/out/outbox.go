package out

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

// OutboxEntry is a row in the transactional outbox - an appended event
// still waiting for the relay to apply it to a projection.
type OutboxEntry struct {
	Sequence domain.Sequence
	Event    domain.ConversationStarted
}

// Outbox is the transactional outbox the relay drains - see
// docs/write-path.md. Enqueue is written alongside EventStore.Append, in
// the same transaction for a real adapter, so a projection can never miss
// an event the store already has.
type Outbox interface {
	Enqueue(ctx context.Context, seq domain.Sequence, event domain.ConversationStarted) error
	Pending(ctx context.Context) ([]OutboxEntry, error)
	MarkDone(ctx context.Context, seq domain.Sequence) error
}
