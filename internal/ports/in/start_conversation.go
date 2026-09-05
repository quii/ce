package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type StartConversationCommand struct {
	ResourceURL *string
	ThreadTitle *string
	Author      *string
	Recipients  *[]string
	Message     *string
}

type StartConversationResult struct {
	ConversationID domain.ConversationID
	Sequence       domain.Sequence
}

type ConversationStarter interface {
	StartConversation(ctx context.Context, cmd StartConversationCommand) (StartConversationResult, error)
}

type startConversationUseCase struct {
	ids    out.IDGenerator
	clock  out.Clock
	events out.EventStore
}

func NewStartConversationUseCase(ids out.IDGenerator, clock out.Clock, events out.EventStore) ConversationStarter {
	return &startConversationUseCase{ids: ids, clock: clock, events: events}
}

func (uc *startConversationUseCase) StartConversation(ctx context.Context, cmd StartConversationCommand) (StartConversationResult, error) {
	conversationID := domain.ConversationID(uc.ids.NewID())

	events, err := domain.StartConversation(domain.StartConversationParams{
		ConversationID: conversationID,
		ThreadID:       domain.ThreadID(uc.ids.NewID()),
		MessageID:      domain.MessageID(uc.ids.NewID()),
		ResourceURL:    cmd.ResourceURL,
		ThreadTitle:    cmd.ThreadTitle,
		Author:         cmd.Author,
		Recipients:     cmd.Recipients,
		Message:        cmd.Message,
		OccurredAt:     uc.clock.Now(),
	})
	if err != nil {
		return StartConversationResult{}, err
	}

	seq, err := uc.events.Append(ctx, events...)
	if err != nil {
		return StartConversationResult{}, err
	}

	return StartConversationResult{ConversationID: conversationID, Sequence: seq}, nil
}
