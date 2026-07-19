package in_test

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/adapters/memory"
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
	if err != nil {
		t.Fatalf("StartConversation returned an unexpected error: %v", err)
	}
}
