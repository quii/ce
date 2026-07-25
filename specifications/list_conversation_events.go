package specifications

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// EventListDriver is the in-port surface a driver has to implement to run
// ListConversationEventsSpecification - see
// docs/adr/0022-specifications-and-drivers.md. Every scenario needs a real
// conversation (and, for some, a reply or an added thread) to list events
// for, the same shape ThreadAddDriver/ThreadReplyDriver already need -
// embedding both rather than inventing a third combination.
type EventListDriver interface {
	ThreadAddDriver
	ThreadReplyDriver
	in.EventLister
}

// wantEvent is exported-field-only so assert.Equal (cmp.Diff under the
// hood) can compare it structurally - cmp panics on unexported fields it
// has no way to access. Only the fields relevant to Type are ever
// populated in a want value, mirroring rule 5's "only the fields relevant
// to that event's type populated".
type wantEvent struct {
	Type        string
	ResourceURL string
	ThreadTitle string
	Author      string
	Recipients  []string
	MessageText string
}

// ListConversationEventsSpecification covers every rule of "list a
// conversation's events": rule 1 (conversation targeted by URL, no body -
// implicit in every scenario, which only ever supplies a conversation id),
// rule 2 (a nonexistent conversation is not found), rule 3 (append order),
// rule 4 (read straight from the event store, visible before the relay
// runs), rule 5 (flat per-event shape, only that type's own fields
// populated), rule 6 (no authorization).
func ListConversationEventsSpecification(t *testing.T, driver EventListDriver) {
	t.Helper()

	t.Run("listing events for a freshly started conversation", func(t *testing.T) {
		started, err := driver.StartConversation(context.Background(), in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{"user-2"},
			Message:     strPtr("Where is my order?"),
		})
		assert.NoErr(t, err, "StartConversation")

		records := listEvents(t, driver, string(started.ConversationID))

		assertEventTypes(t, records, []string{"ConversationCreated", "ThreadStarted", "MessagePosted"})
		assertEvent(t, records[0], wantEvent{Type: "ConversationCreated", ResourceURL: "https://example.com/orders/123"})
		assertEvent(t, records[1], wantEvent{Type: "ThreadStarted", ThreadTitle: "Order query", Author: "user-1", Recipients: []string{"user-2"}})
		assertEvent(t, records[2], wantEvent{Type: "MessagePosted", Author: "user-1", MessageText: "Where is my order?"})
	})

	t.Run("a reply appears as a new event at the end of the list", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		_, err := reply(t, driver, conversationID, threadID, "user-2", "Looking into it")
		assert.NoErr(t, err, "ReplyToThread")

		records := listEvents(t, driver, conversationID)

		assert.Len(t, records, 4, "ListConversationEvents after a reply")
		assertEvent(t, records[3], wantEvent{Type: "MessagePosted", Author: "user-2", MessageText: "Looking into it"})
	})

	t.Run("adding a thread appears as two new events at the end of the list", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		result, err := driver.AddThread(context.Background(), in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr("Delivery date"),
			Author:         strPtr("user-3"),
			Recipients:     &[]string{"user-4"},
			Message:        strPtr("When will this ship?"),
		})
		assert.NoErr(t, err, "AddThread")
		drainAndWait(t, driver, conversationID, result.Sequence)

		records := listEvents(t, driver, conversationID)

		assert.Len(t, records, 5, "ListConversationEvents after AddThread")
		assertEvent(t, records[3], wantEvent{Type: "ThreadStarted", ThreadTitle: "Delivery date", Author: "user-3", Recipients: []string{"user-4"}})
		assertEvent(t, records[4], wantEvent{Type: "MessagePosted", Author: "user-3", MessageText: "When will this ship?"})
	})

	t.Run("listing events for a nonexistent conversation is not found", func(t *testing.T) {
		_, err := driver.ListConversationEvents(context.Background(), in.ListConversationEventsCommand{ConversationID: "missing-conversation"})
		assert.ErrorIs(t, err, domain.ErrConversationNotFound, "ListConversationEvents(missing-conversation)")
	})

	t.Run("events are visible immediately, without waiting for the projection", func(t *testing.T) {
		started, err := driver.StartConversation(context.Background(), in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{"user-2"},
			Message:     strPtr("Where is my order?"),
		})
		assert.NoErr(t, err, "StartConversation")

		records := listEvents(t, driver, string(started.ConversationID))

		assert.Len(t, records, 3, "ListConversationEvents immediately after the write, before Drain")
	})

	t.Run("anyone can list events, even someone not participating in any thread", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		records := listEvents(t, driver, conversationID)

		assert.Len(t, records, 3, "ListConversationEvents for user-99, a non-participant")
	})
}

func listEvents(t *testing.T, driver in.EventLister, conversationID string) []domain.EventRecord {
	t.Helper()

	records, err := driver.ListConversationEvents(context.Background(), in.ListConversationEventsCommand{ConversationID: conversationID})
	assert.NoErr(t, err, "ListConversationEvents(%q)", conversationID)

	return records
}

func assertEventTypes(t *testing.T, records []domain.EventRecord, want []string) {
	t.Helper()

	got := make([]string, len(records))
	for i, record := range records {
		got[i] = record.Event.TypeName()
	}
	assert.Equal(t, got, want, "ListConversationEvents event types")
}

func assertEvent(t *testing.T, record domain.EventRecord, want wantEvent) {
	t.Helper()

	got := wantEvent{Type: record.Event.TypeName()}
	switch e := record.Event.(type) {
	case domain.ConversationCreated:
		got.ResourceURL = string(e.ResourceURL)
	case domain.ThreadStarted:
		got.ThreadTitle = string(e.ThreadTitle)
		got.Author = string(e.Author)
		got.Recipients = make([]string, len(e.Recipients))
		for i, r := range e.Recipients {
			got.Recipients[i] = string(r)
		}
	case domain.MessagePosted:
		got.Author = string(e.Author)
		got.MessageText = string(e.MessageText)
	}

	assert.Equal(t, got, want, "ListConversationEvents record")
}
