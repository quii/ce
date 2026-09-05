package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type AddThreadCommand struct {
	ConversationID string
	ThreadTitle    *string
	Author         *string
	Recipients     *[]string
	Message        *string
}

type AddThreadResult struct {
	ConversationID domain.ConversationID
	Sequence       domain.Sequence
}

type ThreadAdder interface {
	AddThread(ctx context.Context, cmd AddThreadCommand) (AddThreadResult, error)
}

type addThreadUseCase struct {
	ids        out.IDGenerator
	clock      out.Clock
	events     out.EventStore
	projection out.Projection
}

func NewAddThreadUseCase(ids out.IDGenerator, clock out.Clock, events out.EventStore, projection out.Projection) ThreadAdder {
	return &addThreadUseCase{ids: ids, clock: clock, events: events, projection: projection}
}

func (uc *addThreadUseCase) AddThread(ctx context.Context, cmd AddThreadCommand) (AddThreadResult, error) {
	conversationID := domain.ConversationID(cmd.ConversationID)

	events, err := domain.AddThread(domain.AddThreadParams{
		ConversationID: conversationID,
		ThreadID:       domain.ThreadID(uc.ids.NewID()),
		MessageID:      domain.MessageID(uc.ids.NewID()),
		ThreadTitle:    cmd.ThreadTitle,
		Author:         cmd.Author,
		Recipients:     cmd.Recipients,
		Message:        cmd.Message,
		OccurredAt:     uc.clock.Now(),
	})
	if err != nil {
		return AddThreadResult{}, err
	}

	exists, err := uc.projection.Exists(ctx, conversationID)
	if err != nil {
		return AddThreadResult{}, err
	}
	if !exists {
		return AddThreadResult{}, domain.ErrConversationNotFound
	}

	seq, err := uc.events.Append(ctx, events...)
	if err != nil {
		return AddThreadResult{}, err
	}

	return AddThreadResult{ConversationID: conversationID, Sequence: seq}, nil
}
