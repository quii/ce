package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// AddThreadCommand is raw, not-yet-validated input - see
// docs/adr/0010-tiny-types.md. Pointer fields preserve the "omitted" vs
// "present but empty" distinction rule 1 of "add a thread to a
// conversation" depends on. ConversationID comes from the URL, always
// present, so it stays a plain string rather than a pointer.
type AddThreadCommand struct {
	ConversationID string
	ThreadTitle    *string
	Author         *string
	Recipients     *[]string
	Message        *string
}

// AddThreadResult mirrors StartConversationResult/ReplyToThreadResult -
// rule 8 of "add a thread to a conversation" responds with the same shape
// as starting a conversation and replying to a thread.
type AddThreadResult struct {
	ConversationID domain.ConversationID
	Sequence       domain.Sequence
}

type ThreadAdder interface {
	AddThread(ctx context.Context, cmd AddThreadCommand) (AddThreadResult, error)
}

// AddThreadDependencies bundles an addThreadUseCase's out ports into a
// single constructor parameter - see
// docs/adr/0003-commands-not-parameter-lists.md. Projection is here
// (like ReplyToThreadDependencies, unlike StartConversationDependencies)
// because rule 4 needs to check the target conversation actually exists
// before the new thread can be appended.
type AddThreadDependencies struct {
	IDs        out.IDGenerator
	Clock      out.Clock
	Events     out.EventStore
	Projection out.Projection
}

type addThreadUseCase struct {
	deps AddThreadDependencies
}

func NewAddThreadUseCase(deps AddThreadDependencies) ThreadAdder {
	return &addThreadUseCase{deps: deps}
}

func (uc *addThreadUseCase) AddThread(ctx context.Context, cmd AddThreadCommand) (AddThreadResult, error) {
	conversationID := domain.ConversationID(cmd.ConversationID)

	events, err := domain.AddThread(domain.AddThreadParams{
		ConversationID: conversationID,
		ThreadID:       domain.ThreadID(uc.deps.IDs.NewID()),
		MessageID:      domain.MessageID(uc.deps.IDs.NewID()),
		ThreadTitle:    cmd.ThreadTitle,
		Author:         cmd.Author,
		Recipients:     cmd.Recipients,
		Message:        cmd.Message,
		OccurredAt:     uc.deps.Clock.Now(),
	})
	if err != nil {
		return AddThreadResult{}, err
	}

	// Checked only once validation has already passed with no I/O at all -
	// rule 3 of "add a thread to a conversation": a malformed request
	// against a nonexistent conversation is rejected 400, not 404. Exists
	// rather than Get: nothing here needs the conversation's threads or
	// messages, just a yes/no answer.
	exists, err := uc.deps.Projection.Exists(ctx, conversationID)
	if err != nil {
		return AddThreadResult{}, err
	}
	if !exists {
		return AddThreadResult{}, domain.ErrConversationNotFound
	}

	seq, err := uc.deps.Events.Append(ctx, events...)
	if err != nil {
		return AddThreadResult{}, err
	}

	return AddThreadResult{ConversationID: conversationID, Sequence: seq}, nil
}
