package in_test

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// TestStartConversation_OnlyNeedsEventStore is a regression test for a bug
// where StartConversation called Outbox.Enqueue itself, redundantly, after
// Events.Append had already durably written the event and its outbox row
// in one transaction (internal/adapters/postgres/event_store.go, proven by
// the EventStoreEnqueuesViaAppend contract test). A transient failure on
// that second, redundant call surfaced as an error for a conversation that
// was already created - and with no idempotency key anywhere in the API, a
// client retrying on that error would create a duplicate.
//
// StartConversationDependencies has no Outbox field at all now, so this is
// a structural guarantee, not just a runtime one: the use case has no way
// to reach an outbox even if it wanted to.
func TestStartConversation_OnlyNeedsEventStore(t *testing.T) {
	useCase := in.NewStartConversationUseCase(in.StartConversationDependencies{
		IDs:    memory.NewIDGenerator(),
		Clock:  memory.NewClock(),
		Events: memory.NewEventStore(),
	})

	resourceURL, threadTitle, author, message := "https://example.com/orders/123", "Order query", "user-1", "Where is my order?"
	_, err := useCase.StartConversation(context.Background(), in.StartConversationCommand{
		ResourceURL: &resourceURL,
		ThreadTitle: &threadTitle,
		Author:      &author,
		Recipients:  &[]string{},
		Message:     &message,
	})
	assert.NoErr(t, err, "StartConversation")
}

// TestStartConversation_RaisesThreeEventsAtomically proves rule 1 of the
// "conversation event split" story directly against a real
// memory.NewEventStore(): starting a conversation raises ConversationCreated,
// ThreadStarted and MessagePosted together, in one Append call, with
// sequential sequence numbers - see docs/adr/0029-fine-grained-events.md.
func TestStartConversation_RaisesThreeEventsAtomically(t *testing.T) {
	events := memory.NewEventStore()
	useCase := in.NewStartConversationUseCase(in.StartConversationDependencies{
		IDs:    memory.NewIDGenerator(),
		Clock:  memory.NewClock(),
		Events: events,
	})

	resourceURL, threadTitle, author, message := "https://example.com/orders/123", "Order query", "user-1", "Where is my order?"
	result, err := useCase.StartConversation(context.Background(), in.StartConversationCommand{
		ResourceURL: &resourceURL,
		ThreadTitle: &threadTitle,
		Author:      &author,
		Recipients:  &[]string{"user-2"},
		Message:     &message,
	})
	assert.NoErr(t, err, "StartConversation")

	pending, err := events.Pending(context.Background())
	assert.NoErr(t, err, "Pending")
	assert.Len(t, pending, 3, "Pending() after StartConversation")

	wantSequences := []domain.Sequence{1, 2, 3}
	gotSequences := make([]domain.Sequence, len(pending))
	for i, entry := range pending {
		gotSequences[i] = entry.Sequence
	}
	assert.Equal(t, gotSequences, wantSequences, "Pending() sequence order after StartConversation")
	assert.Equal(t, result.Sequence, domain.Sequence(3), "StartConversationResult.Sequence - the last event in the batch")

	created, ok := pending[0].Event.(domain.ConversationCreated)
	if !ok {
		t.Fatalf("Pending()[0].Event = %#v, want a domain.ConversationCreated", pending[0].Event)
	}
	assert.Equal(t, created.ConversationID, result.ConversationID, "Pending()[0].Event.(domain.ConversationCreated).ConversationID")

	threadStarted, ok := pending[1].Event.(domain.ThreadStarted)
	if !ok {
		t.Fatalf("Pending()[1].Event = %#v, want a domain.ThreadStarted", pending[1].Event)
	}
	assert.Equal(t, threadStarted.ConversationID, result.ConversationID, "Pending()[1].Event.(domain.ThreadStarted).ConversationID")

	messagePosted, ok := pending[2].Event.(domain.MessagePosted)
	if !ok {
		t.Fatalf("Pending()[2].Event = %#v, want a domain.MessagePosted", pending[2].Event)
	}
	assert.Equal(t, messagePosted.ConversationID, result.ConversationID, "Pending()[2].Event.(domain.MessagePosted).ConversationID")
	assert.Equal(t, messagePosted.ThreadID, threadStarted.ThreadID, "Pending()[2].Event.(domain.MessagePosted).ThreadID matches the started thread's")
}
