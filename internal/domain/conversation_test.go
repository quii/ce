package domain_test

import (
	"testing"

	"github.com/quii/ce/internal/domain"
)

func TestStartConversation_RecordsAPlaceholderCreator(t *testing.T) {
	resourceURL := "https://example.com/orders/123"
	threadTitle := "Order query"
	author := "user-1"
	recipients := []string{"user-2"}
	message := "Where is my order?"

	event, err := domain.StartConversation(domain.StartConversationParams{
		ConversationID: "conversation-1",
		ThreadID:       "thread-1",
		MessageID:      "message-1",
		ResourceURL:    &resourceURL,
		ThreadTitle:    &threadTitle,
		Author:         &author,
		Recipients:     &recipients,
		Message:        &message,
	})
	if err != nil {
		t.Fatalf("StartConversation returned an unexpected error: %v", err)
	}

	if event.Creator != domain.PlaceholderCreator {
		t.Errorf("Creator = %q, want the fixed placeholder %q - rule 6 defers deriving a real caller identity to a follow-up story", event.Creator, domain.PlaceholderCreator)
	}
}
