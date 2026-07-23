package domain_test

import (
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
)

func TestStartConversation_RecordsAPlaceholderCreator(t *testing.T) {
	resourceURL := "https://example.com/orders/123"
	threadTitle := "Order query"
	author := "user-1"
	recipients := []string{"user-2"}
	message := "Where is my order?"

	events, err := domain.StartConversation(domain.StartConversationParams{
		ConversationID: "conversation-1",
		ThreadID:       "thread-1",
		MessageID:      "message-1",
		ResourceURL:    &resourceURL,
		ThreadTitle:    &threadTitle,
		Author:         &author,
		Recipients:     &recipients,
		Message:        &message,
	})
	assert.NoErr(t, err, "StartConversation")
	assert.Len(t, events, 3, "StartConversation events")

	created, ok := events[0].(domain.ConversationCreated)
	if !ok {
		t.Fatalf("events[0] = %#v, want a domain.ConversationCreated", events[0])
	}
	assert.Equal(t, created.Creator, domain.PlaceholderCreator, "Creator (rule 6 defers deriving a real caller identity to a follow-up story)")
}
