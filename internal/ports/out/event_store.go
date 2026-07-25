package out

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

// EventStore is the append-only log every state change is captured in -
// see docs/adr/0019-event-sourcing-transactional-outbox.md. Append accepts
// a batch: a single logical write can raise more than one event, committed
// atomically in one transaction with sequential sequence numbers
// (docs/adr/0029-fine-grained-events.md) - it returns the sequence of the
// last event in the batch, sufficient for a caller waiting on the whole
// write to have landed, since the earlier events in the same batch are
// guaranteed to have committed first.
//
// ListByConversation reads straight from the log itself, in ascending
// sequence order (append order) - unlike out.Projection, there's no
// checkpoint/catch-up mechanic to wait on here, since the event store is
// the write side, not a read model the relay populates asynchronously. An
// empty result means the conversation has never had an event appended -
// see "list a conversation's events" rule 2.
type EventStore interface {
	Append(ctx context.Context, events ...domain.Event) (domain.Sequence, error)
	ListByConversation(ctx context.Context, id domain.ConversationID) ([]domain.EventRecord, error)
}
