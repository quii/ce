package specifications

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type ThreadParticipantDriver interface {
	ConversationDriver
	in.ThreadAdder
	in.ThreadReplier
	in.ThreadParticipantManager
	in.ConversationsByParticipantGetter
	in.EventLister
	in.Relay
}

func ManageThreadParticipantsSpecification(t *testing.T, driver ThreadParticipantDriver) {
	t.Helper()
	t.Run("adding a participant is scoped to one thread", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "alice", []string{"bob"}, "opening")
		second, err := driver.AddThread(context.Background(), in.AddThreadCommand{ConversationID: conversationID, ThreadTitle: strPtr("Second"), Author: strPtr("dave"), Recipients: &[]string{}, Message: strPtr("second")})
		assert.NoErr(t, err, "AddThread")
		drainAndWait(t, driver, conversationID, second.Sequence)
		result, err := addParticipant(t, driver, conversationID, threadID, "carol")
		assert.NoErr(t, err, "AddThreadParticipant")
		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assert.True(t, result.Changed, "AddThreadParticipant changed")
		assertThreadParticipants(t, view.Threads[0], []string{"alice", "bob", "carol"})
		assertThreadParticipants(t, view.Threads[1], []string{"dave"})
	})

	t.Run("added participant sees history and can reply", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "alice", []string{}, "opening")
		result, err := addParticipant(t, driver, conversationID, threadID, "bob")
		assert.NoErr(t, err, "AddThreadParticipant")
		drainAndWait(t, driver, conversationID, result.Sequence)
		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{ParticipantID: "bob"})
		assert.NoErr(t, err, "GetConversationsByParticipant")
		assertThreadMessages(t, findConversation(t, views, domain.ConversationID(conversationID)).Threads[0], []wantMessage{{Author: "alice", Text: "opening"}})
		replyResult, err := reply(t, driver, conversationID, threadID, "bob", "reply")
		assert.NoErr(t, err, "ReplyToThread by added participant")
		drainAndWait(t, driver, conversationID, replyResult.Sequence)
	})

	t.Run("removing a participant preserves messages and revokes access", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "alice", []string{"bob"}, "opening")
		replyResult, err := reply(t, driver, conversationID, threadID, "bob", "bob reply")
		assert.NoErr(t, err, "ReplyToThread")
		drainAndWait(t, driver, conversationID, replyResult.Sequence)
		result, err := driver.RemoveThreadParticipant(context.Background(), in.ManageThreadParticipantCommand{ConversationID: conversationID, ThreadID: threadID, ParticipantID: "bob"})
		assert.NoErr(t, err, "RemoveThreadParticipant")
		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assertThreadParticipants(t, view.Threads[0], []string{"alice"})
		assertThreadMessages(t, view.Threads[0], []wantMessage{{Author: "alice", Text: "opening"}, {Author: "bob", Text: "bob reply"}})
		views, err := driver.GetConversationsByParticipant(context.Background(), in.GetConversationsByParticipantCommand{ParticipantID: "bob"})
		assert.NoErr(t, err, "GetConversationsByParticipant")
		for _, v := range views {
			assert.True(t, v.ID != domain.ConversationID(conversationID), "removed participant still sees changed conversation")
		}
		_, err = reply(t, driver, conversationID, threadID, "bob", "denied")
		assert.ErrorIs(t, err, domain.ErrReplyForbidden, "ReplyToThread by removed participant")
	})

	t.Run("the final participant can be removed", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "alice", []string{}, "opening")
		result, err := driver.RemoveThreadParticipant(context.Background(), in.ManageThreadParticipantCommand{ConversationID: conversationID, ThreadID: threadID, ParticipantID: "alice"})
		assert.NoErr(t, err, "RemoveThreadParticipant")
		view := drainAndWait(t, driver, conversationID, result.Sequence)
		assert.Len(t, view.Threads[0].Participants, 0, "participants after final removal")
		_, err = reply(t, driver, conversationID, threadID, "alice", "denied")
		assert.ErrorIs(t, err, domain.ErrReplyForbidden, "ReplyToThread after final removal")
	})

	t.Run("repeating membership changes is an immediate no-op", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "alice", []string{}, "opening")
		before, err := driver.ListConversationEvents(context.Background(), in.ListConversationEventsCommand{ConversationID: conversationID})
		assert.NoErr(t, err, "ListConversationEvents before no-op")
		result, err := addParticipant(t, driver, conversationID, threadID, "alice")
		assert.NoErr(t, err, "repeat AddThreadParticipant")
		assert.False(t, result.Changed, "repeat add changed")
		result, err = driver.RemoveThreadParticipant(context.Background(), in.ManageThreadParticipantCommand{ConversationID: conversationID, ThreadID: threadID, ParticipantID: "bob"})
		assert.NoErr(t, err, "repeat RemoveThreadParticipant")
		assert.False(t, result.Changed, "repeat remove changed")
		after, err := driver.ListConversationEvents(context.Background(), in.ListConversationEventsCommand{ConversationID: conversationID})
		assert.NoErr(t, err, "ListConversationEvents after no-op")
		assert.Len(t, after, len(before), "events after no-op")
	})

	t.Run("missing or mismatched threads are not found", func(t *testing.T) {
		_, err := addParticipant(t, driver, "missing", "thread", "alice")
		assertThreadParticipantNotFound(t, err)
		conversationA, _ := startThreadAndCatchUp(t, driver, "alice", []string{}, "opening")
		_, threadB := startThreadAndCatchUp(t, driver, "bob", []string{}, "opening")
		_, err = driver.RemoveThreadParticipant(context.Background(), in.ManageThreadParticipantCommand{ConversationID: conversationA, ThreadID: threadB, ParticipantID: "bob"})
		assertThreadParticipantNotFound(t, err)
	})

	t.Run("membership changes appear in the event history", func(t *testing.T) {
		conversationID, threadID := startThreadAndCatchUp(t, driver, "alice", []string{}, "opening")
		result, err := addParticipant(t, driver, conversationID, threadID, "bob")
		assert.NoErr(t, err, "AddThreadParticipant")
		records := listEvents(t, driver, conversationID)
		assert.Len(t, records, 4, "ListConversationEvents after AddThreadParticipant")
		assertEvent(t, records[3], wantEvent{Type: "ParticipantAdded", ThreadID: threadID, ParticipantID: "bob"})
		drainAndWait(t, driver, conversationID, result.Sequence)
		_, err = driver.RemoveThreadParticipant(context.Background(), in.ManageThreadParticipantCommand{ConversationID: conversationID, ThreadID: threadID, ParticipantID: "bob"})
		assert.NoErr(t, err, "RemoveThreadParticipant")
		records = listEvents(t, driver, conversationID)
		assert.Len(t, records, 5, "ListConversationEvents after RemoveThreadParticipant")
		assertEvent(t, records[4], wantEvent{Type: "ParticipantRemoved", ThreadID: threadID, ParticipantID: "bob"})
	})
}
