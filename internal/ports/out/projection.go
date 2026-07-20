package out

import (
	"context"

	"github.com/quii/ce/internal/domain"
)

// Projection is the read-optimised store GetConversation serves from -
// see docs/adr/0018-cqrs.md. Apply advances both the read model and the
// checkpoint together, so a reader comparing Checkpoint against a
// requested sequence never sees one move without the other.
type Projection interface {
	Apply(ctx context.Context, event domain.Event, seq domain.Sequence) error
	Get(ctx context.Context, id domain.ConversationID) (domain.ConversationView, error)
	Checkpoint(ctx context.Context) (domain.Sequence, error)
}
