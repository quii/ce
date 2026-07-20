package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// ReplyToThreadCommand is raw, not-yet-validated input - see
// docs/adr/0010-tiny-types.md. Author and Message are pointers to
// preserve the "omitted" vs "present but empty" distinction rule 1 of
// "reply to a thread" depends on. ConversationID and ThreadID come from
// the URL, always present, so they stay plain strings rather than
// pointers.
type ReplyToThreadCommand struct {
	ConversationID string
	ThreadID       string
	Author         *string
	Message        *string
}

// ReplyToThreadResult mirrors StartConversationResult - rule 5 of "reply
// to a thread" responds with the same shape as starting a conversation.
type ReplyToThreadResult struct {
	ConversationID domain.ConversationID
	Sequence       domain.Sequence
}

type ThreadReplier interface {
	ReplyToThread(ctx context.Context, cmd ReplyToThreadCommand) (ReplyToThreadResult, error)
}

// ReplyToThreadDependencies bundles a replyToThreadUseCase's out ports
// into a single constructor parameter - see
// docs/adr/0003-commands-not-parameter-lists.md. Projection is here
// (unlike StartConversationDependencies) because rules 2-3 need a
// thread's current participants to check existence and authorship against
// before the reply can be appended.
type ReplyToThreadDependencies struct {
	IDs        out.IDGenerator
	Clock      out.Clock
	Events     out.EventStore
	Projection out.Projection
}

type replyToThreadUseCase struct {
	deps ReplyToThreadDependencies
}

func NewReplyToThreadUseCase(deps ReplyToThreadDependencies) ThreadReplier {
	return &replyToThreadUseCase{deps: deps}
}

func (uc *replyToThreadUseCase) ReplyToThread(ctx context.Context, cmd ReplyToThreadCommand) (ReplyToThreadResult, error) {
	reply, err := domain.ValidateReply(domain.ReplyParams{
		ConversationID: domain.ConversationID(cmd.ConversationID),
		ThreadID:       domain.ThreadID(cmd.ThreadID),
		MessageID:      domain.MessageID(uc.deps.IDs.NewID()),
		Author:         cmd.Author,
		Message:        cmd.Message,
		OccurredAt:     uc.deps.Clock.Now(),
	})
	if err != nil {
		return ReplyToThreadResult{}, err
	}

	view, err := uc.deps.Projection.Get(ctx, reply.ConversationID)
	if err != nil {
		return ReplyToThreadResult{}, err
	}

	if err := domain.AuthorizeReply(view, reply); err != nil {
		return ReplyToThreadResult{}, err
	}

	seq, err := uc.deps.Events.Append(ctx, reply)
	if err != nil {
		return ReplyToThreadResult{}, err
	}

	return ReplyToThreadResult{ConversationID: reply.ConversationID, Sequence: seq}, nil
}
