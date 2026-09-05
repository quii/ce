package specifications

import (
	"context"
	"errors"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

func addParticipant(t *testing.T, driver in.ThreadParticipantManager, conversationID, threadID, participantID string) (in.ManageThreadParticipantResult, error) {
	t.Helper()
	return driver.AddThreadParticipant(context.Background(), in.ManageThreadParticipantCommand{ConversationID: conversationID, ThreadID: threadID, ParticipantID: participantID})
}

func assertThreadParticipants(t *testing.T, thread domain.ThreadView, want []string) {
	t.Helper()
	assert.Len(t, thread.Participants, len(want), "Thread.Participants")
	for _, id := range want {
		assert.Contains(t, thread.Participants, domain.ParticipantID(id), "Thread.Participants")
	}
}

func assertThreadParticipantNotFound(t *testing.T, err error) {
	t.Helper()
	assert.True(t, errors.Is(err, domain.ErrConversationNotFound) || errors.Is(err, domain.ErrThreadNotFound), "ManageThreadParticipant returned err = %v, want domain.ErrConversationNotFound or domain.ErrThreadNotFound", err)
}
