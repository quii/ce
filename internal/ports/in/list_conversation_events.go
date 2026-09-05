package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

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
