package specifications

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// ThreadReplyDriver is the in-port surface a driver has to implement to
// run ReplyToThreadSpecification - see docs/adr/0022-specifications-and-drivers.md.
// Every scenario needs a real, already-projected conversation to reply
// against - rules 2-3 look the thread's current participants up via the
// projection before a reply can be authorized - so this driver always
// needs in.Relay too, unlike the plainer ConversationDriver.
type ThreadReplyDriver interface {
	ConversationDriver
	in.ThreadReplier
	in.Relay
}

// wantMessage is exported-field-only so assert.Equal (cmp.Diff under the
// hood) can compare it structurally - cmp panics on unexported fields it
// has no way to access.
type wantMessage struct {
	Author string
	Text   string
}

// ReplyToThreadSpecification covers every rule of "reply to a thread":
// rule 1 (required fields), rule 2 (existence), rule 3 (authorship),
// rule 4 (check ordering), rule 5 (202 + Location, unaltered title/
// recipients), rule 6 (server-set timestamp), rule 7 (append order), and
// rule 8 (pending/plain-read semantics, unchanged from "start a
// conversation").
func ReplyToThreadSpecification(t *testing.T, driver ThreadReplyDriver) {
	t.Helper()

	t.Run("the thread's original author replies", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		reply, err := reply(t, driver, conversationID, threadID, "user-1", "Let me know when you can")
		assert.NoErr(t, err, "ReplyToThread")

		view := drainAndWait(t, driver, conversationID, reply.Sequence)
		assertMessagesInOrder(t, httpConversationView{view}, []wantMessage{
			{Author: "user-1", Text: "Where is my order?"},
			{Author: "user-1", Text: "Let me know when you can"},
		})
	})

	t.Run("a recipient replies", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		reply, err := reply(t, driver, conversationID, threadID, "user-2", "Looking into it")
		assert.NoErr(t, err, "ReplyToThread")

		view := drainAndWait(t, driver, conversationID, reply.Sequence)
		assertMessagesInOrder(t, httpConversationView{view}, []wantMessage{
			{Author: "user-1", Text: "Where is my order?"},
			{Author: "user-2", Text: "Looking into it"},
		})
	})

	t.Run("someone outside the thread's participants is forbidden from replying", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		_, err := reply(t, driver, conversationID, threadID, "user-3", "Can I help?")
		assert.ErrorIs(t, err, domain.ErrReplyForbidden, "ReplyToThread from a non-participant")
	})

	t.Run("replying to a nonexistent conversation is not found", func(t *testing.T) {
		_, err := reply(t, driver, "missing-conversation", "missing-thread", "user-1", "hello?")
		assertReplyNotFound(t, err)
	})

	t.Run("replying with a thread id that doesn't belong to the given conversation is not found", func(t *testing.T) {
		conversationA, _ := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "About conversation A")
		_, threadB := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "About conversation B")

		_, err := reply(t, driver, conversationA, threadB, "user-1", "wrong thread")
		assertReplyNotFound(t, err)
	})

	t.Run("a missing author is rejected", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		_, err := driver.ReplyToThread(context.Background(), in.ReplyToThreadCommand{
			ConversationID: conversationID,
			ThreadID:       threadID,
			Message:        strPtr("no author here"),
		})
		assertReplyRejected(t, err, "author is required")
	})

	t.Run("missing message text is rejected", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		_, err := driver.ReplyToThread(context.Background(), in.ReplyToThreadCommand{
			ConversationID: conversationID,
			ThreadID:       threadID,
			Author:         strPtr("user-1"),
		})
		assertReplyRejected(t, err, "message is required")
	})

	t.Run("empty string message text is accepted", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		result, err := reply(t, driver, conversationID, threadID, "user-1", "")
		assert.NoErr(t, err, "ReplyToThread")

		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assertMessagesInOrder(t, httpConversationView{view}, []wantMessage{
			{Author: "user-1", Text: "Where is my order?"},
			{Author: "user-1", Text: ""},
		})
	})

	t.Run("a malformed request to a nonexistent thread is rejected as a bad request, not a not-found", func(t *testing.T) {
		_, err := driver.ReplyToThread(context.Background(), in.ReplyToThreadCommand{
			ConversationID: "missing-conversation",
			ThreadID:       "missing-thread",
			Message:        strPtr("no author here"),
		})
		assertReplyRejected(t, err, "author is required")
	})

	t.Run("replies land in posting order", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		first, err := reply(t, driver, conversationID, threadID, "user-1", "first reply")
		assert.NoErr(t, err, "ReplyToThread")
		drainAndWait(t, driver, conversationID, first.Sequence)

		second, err := reply(t, driver, conversationID, threadID, "user-2", "second reply")
		assert.NoErr(t, err, "ReplyToThread")
		view := drainAndWait(t, driver, conversationID, second.Sequence)

		assertMessagesInOrder(t, httpConversationView{view}, []wantMessage{
			{Author: "user-1", Text: "Where is my order?"},
			{Author: "user-1", Text: "first reply"},
			{Author: "user-2", Text: "second reply"},
		})
	})

	t.Run("a reply is pending until the projection catches up", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "user-1", []string{"user-2"}, "Where is my order?")

		result, err := reply(t, driver, conversationID, threadID, "user-1", "Let me know when you can")
		assert.NoErr(t, err, "ReplyToThread")

		after := int64(result.Sequence)
		_, err = driver.GetConversation(context.Background(), in.GetConversationCommand{
			ConversationID: conversationID,
			After:          &after,
		})
		assert.ErrorIs(t, err, domain.ErrProjectionNotCaughtUp, "GetConversation(after=%d) immediately after the reply", after)

		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assertMessagesInOrder(t, httpConversationView{view}, []wantMessage{
			{Author: "user-1", Text: "Where is my order?"},
			{Author: "user-1", Text: "Let me know when you can"},
		})
	})
}
