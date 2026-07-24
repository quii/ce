package domain_test

import (
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
)

// TestAddThread_RaisesThreadStartedAndMessagePosted proves rule 6 of "add
// a thread to a conversation" directly against the pure domain function:
// adding a thread raises exactly a ThreadStarted and a MessagePosted, in
// that order, with no ConversationCreated - the conversation already
// exists, so there's nothing to create.
func TestAddThread_RaisesThreadStartedAndMessagePosted(t *testing.T) {
	threadTitle := "Delivery date"
	author := "user-3"
	recipients := []string{"user-4"}
	message := "When will this ship?"

	events, err := domain.AddThread(domain.AddThreadParams{
		ConversationID: "conversation-1",
		ThreadID:       "thread-2",
		MessageID:      "message-2",
		ThreadTitle:    &threadTitle,
		Author:         &author,
		Recipients:     &recipients,
		Message:        &message,
	})
	assert.NoErr(t, err, "AddThread")
	assert.Len(t, events, 2, "AddThread events")

	threadStarted, ok := events[0].(domain.ThreadStarted)
	if !ok {
		t.Fatalf("events[0] = %#v, want a domain.ThreadStarted", events[0])
	}
	assert.Equal(t, threadStarted.ThreadTitle, domain.ThreadTitle(threadTitle), "ThreadStarted.ThreadTitle")

	messagePosted, ok := events[1].(domain.MessagePosted)
	if !ok {
		t.Fatalf("events[1] = %#v, want a domain.MessagePosted", events[1])
	}
	assert.Equal(t, messagePosted.ThreadID, threadStarted.ThreadID, "MessagePosted.ThreadID matches the started thread's")
}
