package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// ListConversationEventsCommand is raw, not-yet-validated input.
// ConversationID comes from the URL, always present, so it stays a plain
// string rather than a pointer - see docs/adr/0010-tiny-types.md.
type ListConversationEventsCommand struct {
	ConversationID string
}

type EventLister interface {
	ListConversationEvents(ctx context.Context, cmd ListConversationEventsCommand) ([]domain.EventRecord, error)
}

type listConversationEventsUseCase struct {
	events out.EventStore
}

func NewListConversationEventsUseCase(events out.EventStore) EventLister {
	return &listConversationEventsUseCase{events: events}
}

// ListConversationEvents reads straight from out.EventStore, never
// out.Projection - "list a conversation's events" rule 4: there's no
// pending/checkpoint mechanic to wait on here, only 200 or 404. Rule 2:
// existence is derived from the result itself - a conversation with no
// events is indistinguishable from one that never existed, since every
// conversation's first-ever write is a ConversationCreated event, so
// there's no separate existence check to perform.
func (uc *listConversationEventsUseCase) ListConversationEvents(ctx context.Context, cmd ListConversationEventsCommand) ([]domain.EventRecord, error) {
	records, err := uc.events.ListByConversation(ctx, domain.ConversationID(cmd.ConversationID))
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, domain.ErrConversationNotFound
	}

	return records, nil
}
