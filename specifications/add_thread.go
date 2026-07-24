package specifications

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// ThreadAddDriver is the in-port surface a driver has to implement to run
// AddThreadSpecification - see docs/adr/0022-specifications-and-drivers.md.
// Every scenario needs a real, already-projected conversation to add a
// thread to, the same shape ThreadReplyDriver already needs.
type ThreadAddDriver interface {
	ConversationDriver
	in.ThreadAdder
	in.Relay
}

// AddThreadSpecification covers every rule of "add a thread to a
// conversation": rule 1 (required fields), rule 2 (recipients set/author
// exclusion), rule 3 (check ordering), rule 4 (existence), rule 5 (no
// authorization - anyone can add a thread), rule 6 (ThreadStarted +
// MessagePosted raised, no ConversationCreated), rule 7 (participants
// union, frozen - already proven independently by "thread participants",
// exercised here only incidentally), rule 8 (202 + Location), rule 9 (the
// representation widens to a list of threads), rule 10 (creation order),
// rule 11 (pending/plain-read semantics, unchanged), rule 12 (no limit, no
// uniqueness check - implicit in every scenario adding more than one
// thread with no rejection).
func AddThreadSpecification(t *testing.T, driver ThreadAddDriver) {
	t.Helper()

	validCommand := func(conversationID string) in.AddThreadCommand {
		return in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr("Delivery date"),
			Author:         strPtr("user-3"),
			Recipients:     &[]string{"user-4"},
			Message:        strPtr("When will this ship?"),
		}
	}

	t.Run("a thread is added to an existing conversation with all required fields", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		result, err := driver.AddThread(context.Background(), validCommand(conversationID))
		assert.NoErr(t, err, "AddThread")

		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assert.Len(t, view.Threads, 2, "Threads after AddThread")
		assert.Equal(t, string(view.Threads[0].Title), "Order query", "Threads[0].Title")
		assert.Equal(t, string(view.Threads[1].Title), "Delivery date", "Threads[1].Title")

		assertThreadParticipants(t, view.Threads[1], []string{"user-3", "user-4"})
		assertThreadMessages(t, view.Threads[1], []wantMessage{{Author: "user-3", Text: "When will this ship?"}})
	})

	t.Run("empty string values are accepted", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr(""),
			Author:         strPtr("user-1"),
			Recipients:     &[]string{},
			Message:        strPtr(""),
		}
		result, err := driver.AddThread(context.Background(), cmd)
		assert.NoErr(t, err, "AddThread")

		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assert.Len(t, view.Threads, 2, "Threads after AddThread")
		assert.Equal(t, string(view.Threads[1].Title), "", "Threads[1].Title")
		assertThreadMessages(t, view.Threads[1], []wantMessage{{Author: "user-1", Text: ""}})
	})

	t.Run("a missing thread title is rejected", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := validCommand(conversationID)
		cmd.ThreadTitle = nil
		assertAddThreadRejected(t, driver, cmd, "threadTitle is required")
	})

	t.Run("a missing author is rejected", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := validCommand(conversationID)
		cmd.Author = nil
		assertAddThreadRejected(t, driver, cmd, "author is required")
	})

	t.Run("a missing opening message is rejected", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := validCommand(conversationID)
		cmd.Message = nil
		assertAddThreadRejected(t, driver, cmd, "message is required")
	})

	t.Run("a missing recipients field is rejected", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := validCommand(conversationID)
		cmd.Recipients = nil
		assertAddThreadRejected(t, driver, cmd, "recipients is required")
	})

	t.Run("duplicate recipient ids are rejected", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := validCommand(conversationID)
		cmd.Recipients = &[]string{"user-3", "user-3"}
		assertAddThreadRejected(t, driver, cmd, "recipients must not contain a duplicate id")
	})

	t.Run("an author who is also listed as a recipient is rejected", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := validCommand(conversationID)
		cmd.Author = strPtr("user-3")
		cmd.Recipients = &[]string{"user-3", "user-4"}
		assertAddThreadRejected(t, driver, cmd, "author must not also appear in recipients")
	})

	t.Run("adding a thread to a nonexistent conversation is not found", func(t *testing.T) {
		_, err := driver.AddThread(context.Background(), validCommand("missing-conversation"))
		assert.ErrorIs(t, err, domain.ErrConversationNotFound, "AddThread against a nonexistent conversation")
	})

	t.Run("a malformed request to a nonexistent conversation is rejected as a bad request, not a not-found", func(t *testing.T) {
		cmd := validCommand("missing-conversation")
		cmd.Author = nil
		assertAddThreadRejected(t, driver, cmd, "author is required")
	})

	t.Run("anyone can add a thread, even someone not already participating in the conversation", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		cmd := in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr("A new topic"),
			Author:         strPtr("user-99"),
			Recipients:     &[]string{},
			Message:        strPtr("hello"),
		}
		_, err := driver.AddThread(context.Background(), cmd)
		assert.NoErr(t, err, "AddThread from a non-participant of the existing thread")
	})

	t.Run("threads appear in creation order", func(t *testing.T) {
		started, err := driver.StartConversation(context.Background(), in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("First topic"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{"user-2"},
			Message:     strPtr("first"),
		})
		assert.NoErr(t, err, "StartConversation")
		conversationID := string(started.ConversationID)
		drainAndWait(t, driver, conversationID, started.Sequence)

		second, err := driver.AddThread(context.Background(), in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr("Second topic"),
			Author:         strPtr("user-1"),
			Recipients:     &[]string{},
			Message:        strPtr("second"),
		})
		assert.NoErr(t, err, "AddThread (second topic)")
		drainAndWait(t, driver, conversationID, second.Sequence)

		third, err := driver.AddThread(context.Background(), in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr("Third topic"),
			Author:         strPtr("user-1"),
			Recipients:     &[]string{},
			Message:        strPtr("third"),
		})
		assert.NoErr(t, err, "AddThread (third topic)")
		view := drainAndWait(t, driver, conversationID, third.Sequence)

		assert.Len(t, view.Threads, 3, "Threads after adding two more")
		wantTitles := []string{"First topic", "Second topic", "Third topic"}
		for i, want := range wantTitles {
			assert.Equal(t, string(view.Threads[i].Title), want, "Threads[%d].Title", i)
		}
	})

	t.Run("adding a thread is pending until the projection catches up", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		result, err := driver.AddThread(context.Background(), validCommand(conversationID))
		assert.NoErr(t, err, "AddThread")

		after := int64(result.Sequence)
		_, err = driver.GetConversation(context.Background(), in.GetConversationCommand{
			ConversationID: conversationID,
			After:          &after,
		})
		assert.ErrorIs(t, err, domain.ErrProjectionNotCaughtUp, "GetConversation(after=%d) immediately after AddThread", after)

		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assert.Len(t, view.Threads, 2, "Threads after the projection has caught up")
	})
}

// assertAddThreadRejected checks both that AddThread was rejected as a
// domain.ValidationError and that the message it carries is the real
// server-generated one (wantMessage), not a driver's generic fallback
// text - mirrors assertReplyRejected (specifications/reply_helpers.go).
func assertAddThreadRejected(t *testing.T, driver ThreadAddDriver, cmd in.AddThreadCommand, wantMessage string) {
	t.Helper()

	_, err := driver.AddThread(context.Background(), cmd)
	validationErr := assert.ErrorAs[domain.ValidationError](t, err, "AddThread(%+v)", cmd)
	assert.Equal(t, validationErr.Error(), wantMessage, "AddThread error message")
}
