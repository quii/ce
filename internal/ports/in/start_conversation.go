package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// StartConversationCommand is raw, not-yet-validated input - see
// docs/adr/0010-tiny-types.md. Pointer fields preserve the distinction
// between "omitted" and "present but empty", which rule 2 of the "start a
// conversation" story depends on.
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

// StartConversationDependencies bundles a startConversationUseCase's out
// ports into a single constructor parameter - see
// docs/adr/0003-commands-not-parameter-lists.md.
//
// Outbox is deliberately not one of them: Events.Append is solely
// responsible for durably writing both the event and its outbox row, in
// one transaction (internal/adapters/postgres/event_store.go, proven by
// the EventStoreEnqueuesViaAppend contract test). A second, separate
// Outbox.Enqueue call here would be redundant at best - and at worst, a
// transient failure on that second call would surface as an error for a
// conversation that was already durably created, pushing a retrying
// caller (with no idempotency key anywhere in the API) into creating a
// duplicate.
type StartConversationDependencies struct {
	IDs    out.IDGenerator
	Clock  out.Clock
	Events out.EventStore
}

type startConversationUseCase struct {
	deps StartConversationDependencies
}

func NewStartConversationUseCase(deps StartConversationDependencies) ConversationStarter {
	return &startConversationUseCase{deps: deps}
}

func (uc *startConversationUseCase) StartConversation(ctx context.Context, cmd StartConversationCommand) (StartConversationResult, error) {
	conversationID := domain.ConversationID(uc.deps.IDs.NewID())

	events, err := domain.StartConversation(domain.StartConversationParams{
		ConversationID: conversationID,
		ThreadID:       domain.ThreadID(uc.deps.IDs.NewID()),
		MessageID:      domain.MessageID(uc.deps.IDs.NewID()),
		ResourceURL:    cmd.ResourceURL,
		ThreadTitle:    cmd.ThreadTitle,
		Author:         cmd.Author,
		Recipients:     cmd.Recipients,
		Message:        cmd.Message,
		OccurredAt:     uc.deps.Clock.Now(),
	})
	if err != nil {
		return StartConversationResult{}, err
	}

	// events raises ConversationCreated, ThreadStarted and MessagePosted
	// together - Append commits all three atomically in one write, per
	// docs/adr/0029-fine-grained-events.md.
	seq, err := uc.deps.Events.Append(ctx, events...)
	if err != nil {
		return StartConversationResult{}, err
	}

	return StartConversationResult{ConversationID: conversationID, Sequence: seq}, nil
}
