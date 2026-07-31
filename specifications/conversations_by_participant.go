package specifications

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// ConversationsByParticipantDriver is the in-port surface a driver has to
// implement to run ConversationsByParticipantSpecification - see
// docs/adr/0022-specifications-and-drivers.md.
type ConversationsByParticipantDriver interface {
	in.ConversationStarter
	in.ConversationGetter
	in.ThreadAdder
	in.ThreadReplier
	in.ConversationsByParticipantGetter
	in.Relay
}

// ConversationsByParticipantSpecification covers the "get conversations by
// participant" story's seven scenarios.
func ConversationsByParticipantSpecification(t *testing.T, driver ConversationsByParticipantDriver) {
	t.Helper()

	t.Run("participant on one thread of a multi-thread conversation", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "part-user-1", []string{}, "Hello")

		addResult, err := driver.AddThread(context.Background(), in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr("Thread 2"),
			Author:         strPtr("other-user-1"),
			Recipients:     &[]string{},
			Message:        strPtr("Other thread"),
		})
		assert.NoErr(t, err, "AddThread")
		drainAndWait(t, driver, conversationID, addResult.Sequence)

		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{
			ParticipantID: "part-user-1",
		})
		assert.NoErr(t, err, "GetConversationsByParticipant")

		conv := findConversation(t, views, domain.ConversationID(conversationID))
		assert.Len(t, conv.Threads, 1, "only thread 1 for part-user-1")
		assert.Equal(t, string(conv.Threads[0].Title), "Order query", "thread title")
	})

	t.Run("participant on no threads of a conversation is omitted", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "other-user-2", []string{}, "Hi")

		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{
			ParticipantID: "non-participant-2",
		})
		assert.NoErr(t, err, "GetConversationsByParticipant")

		ids := make([]domain.ConversationID, len(views))
		for i, v := range views {
			ids[i] = v.ID
		}
		assert.NotContains(t, ids, domain.ConversationID(conversationID), "conversations returned for non-participant")
	})

	t.Run("participant appears in multiple conversations", func(t *testing.T) {
		conversationA, _ := startThreadAndCatchUp(t, driver, "multi-conv-user", []string{}, "In A")
		conversationB, _ := startThreadAndCatchUp(t, driver, "multi-conv-user", []string{}, "In B")

		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{
			ParticipantID: "multi-conv-user",
		})
		assert.NoErr(t, err, "GetConversationsByParticipant")

		convA := findConversation(t, views, domain.ConversationID(conversationA))
		assert.Len(t, convA.Threads, 1, "convA thread count")

		convB := findConversation(t, views, domain.ConversationID(conversationB))
		assert.Len(t, convB.Threads, 1, "convB thread count")
	})

	t.Run("results ordered by most recently active thread", func(t *testing.T) {
		conversationA, _ := startThreadAndCatchUp(t, driver, "order-user", []string{}, "Posted first")
		conversationB, _ := startThreadAndCatchUp(t, driver, "order-user", []string{}, "Posted second")

		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{
			ParticipantID: "order-user",
		})
		assert.NoErr(t, err, "GetConversationsByParticipant")

		idxA := conversationIndex(t, views, domain.ConversationID(conversationA))
		idxB := conversationIndex(t, views, domain.ConversationID(conversationB))
		assert.True(t, idxB < idxA, "conversation B (newer) should appear before conversation A (older): idxB=%d idxA=%d", idxB, idxA)
	})

	t.Run("a reply bumps a conversation up the list", func(t *testing.T) {
		conversationA, threadA := startThreadAndCatchUp(t, driver, "bump-user", []string{}, "Conv A message")
		conversationB, _ := startThreadAndCatchUp(t, driver, "bump-user", []string{}, "Conv B message")

		replyResult, err := reply(t, driver, conversationA, threadA, "bump-user", "Reply bumping A")
		assert.NoErr(t, err, "ReplyToThread")
		drainAndWait(t, driver, conversationA, replyResult.Sequence)

		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{
			ParticipantID: "bump-user",
		})
		assert.NoErr(t, err, "GetConversationsByParticipant")

		idxA := conversationIndex(t, views, domain.ConversationID(conversationA))
		idxB := conversationIndex(t, views, domain.ConversationID(conversationB))
		assert.True(t, idxA < idxB, "conversation A (bumped by reply) should appear before conversation B: idxA=%d idxB=%d", idxA, idxB)
	})

	t.Run("participant on multiple threads within one conversation", func(t *testing.T) {
		conversationID, _ := startThreadAndCatchUp(t, driver, "multi-thread-user", []string{}, "Thread 1 message")

		addResult, err := driver.AddThread(context.Background(), in.AddThreadCommand{
			ConversationID: conversationID,
			ThreadTitle:    strPtr("Second thread"),
			Author:         strPtr("multi-thread-user"),
			Recipients:     &[]string{},
			Message:        strPtr("Thread 2 message"),
		})
		assert.NoErr(t, err, "AddThread")
		drainAndWait(t, driver, conversationID, addResult.Sequence)

		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{
			ParticipantID: "multi-thread-user",
		})
		assert.NoErr(t, err, "GetConversationsByParticipant")

		conv := findConversation(t, views, domain.ConversationID(conversationID))
		assert.Len(t, conv.Threads, 2, "both threads for multi-thread-user")
	})

	t.Run("no conversations exist for participant returns empty list", func(t *testing.T) {
		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{
			ParticipantID: "nobody-unique-xyz987",
		})
		assert.NoErr(t, err, "GetConversationsByParticipant")
		assert.Len(t, views, 0, "empty list for unknown participant")
	})
}

func findConversation(t *testing.T, views []domain.ConversationView, id domain.ConversationID) domain.ConversationView {
	t.Helper()

	for _, v := range views {
		if v.ID == id {
			return v
		}
	}
	t.Fatalf("conversation %s not found in results", id)
	return domain.ConversationView{}
}

func conversationIndex(t *testing.T, views []domain.ConversationView, id domain.ConversationID) int {
	t.Helper()

	for i, v := range views {
		if v.ID == id {
			return i
		}
	}
	t.Fatalf("conversation %s not found in results", id)
	return -1
}
