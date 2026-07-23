package out

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

// Projection is the read-optimised store GetConversation serves from -
// see docs/adr/0018-cqrs.md. Apply accepts a batch of OutboxEntry (the
// same pairing of event and sequence out.Outbox.Pending returns) and
// applies every one of them before advancing the checkpoint once - not
// once per event - so a reader comparing Checkpoint against a requested
// sequence never observes a batch partially applied (e.g. a ThreadStarted
// with no MessagePosted companion yet), per
// docs/adr/0029-fine-grained-events.md.
type Projection interface {
	Apply(ctx context.Context, entries ...OutboxEntry) error
	Get(ctx context.Context, id domain.ConversationID) (domain.ConversationView, error)
	Checkpoint(ctx context.Context) (domain.Sequence, error)
}
