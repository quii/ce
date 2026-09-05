package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type ReplyToThreadCommand struct {
	ConversationID string
	ThreadID       string
	Author         *string
	Message        *string
}

type ReplyToThreadResult struct {
	ConversationID domain.ConversationID
	Sequence       domain.Sequence
}

type ThreadReplier interface {
	ReplyToThread(ctx context.Context, cmd ReplyToThreadCommand) (ReplyToThreadResult, error)
}

type replyToThreadUseCase struct {
	ids        out.IDGenerator
	clock      out.Clock
	events     out.EventStore
	projection out.Projection
}

func NewReplyToThreadUseCase(ids out.IDGenerator, clock out.Clock, events out.EventStore, projection out.Projection) ThreadReplier {
	return &replyToThreadUseCase{ids: ids, clock: clock, events: events, projection: projection}
}

func (uc *replyToThreadUseCase) ReplyToThread(ctx context.Context, cmd ReplyToThreadCommand) (ReplyToThreadResult, error) {
	reply, err := domain.ValidateReply(domain.ReplyParams{
		ConversationID: domain.ConversationID(cmd.ConversationID),
		ThreadID:       domain.ThreadID(cmd.ThreadID),
		MessageID:      domain.MessageID(uc.ids.NewID()),
		Author:         cmd.Author,
		Message:        cmd.Message,
		OccurredAt:     uc.clock.Now(),
	})
	if err != nil {
		return ReplyToThreadResult{}, err
	}

	view, err := uc.projection.Get(ctx, reply.ConversationID)
	if err != nil {
		return ReplyToThreadResult{}, err
	}

	if err := domain.AuthorizeReply(view, reply); err != nil {
		return ReplyToThreadResult{}, err
	}

	seq, err := uc.events.Append(ctx, reply)
	if err != nil {
		return ReplyToThreadResult{}, err
	}

	return ReplyToThreadResult{ConversationID: reply.ConversationID, Sequence: seq}, nil
}
