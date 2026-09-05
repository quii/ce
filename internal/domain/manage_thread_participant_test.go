package domain_test

import (
	"testing"
	"time"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
)

func TestConversationAddParticipant(t *testing.T) {
	conversation := rehydrateWithSingleThread(t, "conversation-1", "thread-1", "alice")
	params := domain.ManageThreadParticipantParams{ConversationID: "conversation-1", ThreadID: "thread-1", ParticipantID: "bob", OccurredAt: time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)}

	event, changed, err := conversation.AddParticipant(params)
	assert.NoErr(t, err, "AddParticipant(absent participant)")
	assert.True(t, changed, "AddParticipant(absent participant) changed")
	assert.Equal(t, event, domain.Event(domain.ParticipantAdded{ConversationID: "conversation-1", ThreadID: "thread-1", ParticipantID: "bob", OccurredAt: params.OccurredAt}), "AddParticipant(absent participant) event")

	_, changed, err = conversation.AddParticipant(domain.ManageThreadParticipantParams{ConversationID: "conversation-1", ThreadID: "thread-1", ParticipantID: "alice"})
	assert.NoErr(t, err, "AddParticipant(existing participant)")
	assert.False(t, changed, "AddParticipant(existing participant) changed")
}

func TestConversationRemoveParticipant(t *testing.T) {
	conversation := rehydrateWithSingleThread(t, "conversation-1", "thread-1", "alice")

	event, changed, err := conversation.RemoveParticipant(domain.ManageThreadParticipantParams{ConversationID: "conversation-1", ThreadID: "thread-1", ParticipantID: "alice"})
	assert.NoErr(t, err, "RemoveParticipant(existing participant)")
	assert.True(t, changed, "RemoveParticipant(existing participant) changed")
	assert.Equal(t, event, domain.Event(domain.ParticipantRemoved{ConversationID: "conversation-1", ThreadID: "thread-1", ParticipantID: "alice"}), "RemoveParticipant(existing participant) event")

	_, changed, err = conversation.RemoveParticipant(domain.ManageThreadParticipantParams{ConversationID: "conversation-1", ThreadID: "thread-1", ParticipantID: "bob"})
	assert.NoErr(t, err, "RemoveParticipant(absent participant)")
	assert.False(t, changed, "RemoveParticipant(absent participant) changed")
}

func TestConversationAddOrRemoveParticipantOnUnknownThread(t *testing.T) {
	conversation := rehydrateWithSingleThread(t, "conversation-1", "thread-1", "alice")

	_, _, err := conversation.AddParticipant(domain.ManageThreadParticipantParams{ThreadID: "does-not-exist", ParticipantID: "bob"})
	assert.ErrorIs(t, err, domain.ErrThreadNotFound, "AddParticipant(unknown thread)")

	_, _, err = conversation.RemoveParticipant(domain.ManageThreadParticipantParams{ThreadID: "does-not-exist", ParticipantID: "alice"})
	assert.ErrorIs(t, err, domain.ErrThreadNotFound, "RemoveParticipant(unknown thread)")
}

func TestRehydrateConversationRejectsMissingConversation(t *testing.T) {
	_, err := domain.RehydrateConversation(nil)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound, "RehydrateConversation(no events)")
}

func rehydrateWithSingleThread(t *testing.T, conversationID domain.ConversationID, threadID domain.ThreadID, author domain.ParticipantID) domain.Conversation {
	t.Helper()
	conversation, err := domain.RehydrateConversation([]domain.EventRecord{
		{Event: domain.ConversationCreated{ConversationID: conversationID}},
		{Event: domain.ThreadStarted{ConversationID: conversationID, ThreadID: threadID, Author: author}},
	})
	assert.NoErr(t, err, "RehydrateConversation")
	return conversation
}
