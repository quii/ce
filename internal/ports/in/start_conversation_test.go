package in_test

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

func TestStartConversation_OnlyNeedsEventStore(t *testing.T) {
	useCase := in.NewStartConversationUseCase(memory.NewIDGenerator(), memory.NewClock(), memory.NewEventStore())

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

func TestStartConversation_RaisesThreeEventsAtomically(t *testing.T) {
	events := memory.NewEventStore()
	useCase := in.NewStartConversationUseCase(memory.NewIDGenerator(), memory.NewClock(), events)

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
