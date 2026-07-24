package specifications

import (
	"context"
	"errors"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// conversationRelayDriver is the minimal shape drainAndWait/
// startThreadAndCatchUp need - a real, already-projected conversation to
// target, regardless of which specific in-port (ThreadReplier, ThreadAdder)
// a given specification is otherwise driving. Both ThreadReplyDriver and
// ThreadAddDriver satisfy it.
type conversationRelayDriver interface {
	ConversationDriver
	in.Relay
}

func startThreadAndCatchUp(t *testing.T, driver conversationRelayDriver, author string, recipients []string, message string) (conversationID, threadID string) {
	t.Helper()
	ctx := context.Background()

	started, err := driver.StartConversation(ctx, in.StartConversationCommand{
		ResourceURL: strPtr("https://example.com/orders/123"),
		ThreadTitle: strPtr("Order query"),
		Author:      strPtr(author),
		Recipients:  &recipients,
		Message:     strPtr(message),
	})
	assert.NoErr(t, err, "StartConversation")

	view := drainAndWait(t, driver, string(started.ConversationID), started.Sequence)

	return string(started.ConversationID), string(view.Threads[0].ID)
}

func reply(t *testing.T, driver ThreadReplyDriver, conversationID, threadID, author, message string) (in.ReplyToThreadResult, error) {
	t.Helper()

	return driver.ReplyToThread(context.Background(), in.ReplyToThreadCommand{
		ConversationID: conversationID,
		ThreadID:       threadID,
		Author:         strPtr(author),
		Message:        strPtr(message),
	})
}

func drainAndWait(t *testing.T, driver conversationRelayDriver, conversationID string, seq domain.Sequence) domain.ConversationView {
	t.Helper()
	ctx := context.Background()

	assert.NoErr(t, driver.Drain(ctx), "Drain")

	return waitForProjection(t, driver, conversationID, int64(seq))
}

// assertReplyRejected checks both that ReplyToThread was rejected as a
// domain.ValidationError and that the message it carries is the real
// server-generated one (wantMessage), not a driver's generic fallback
// text - see internal/adapters/httpapi/conversation_handler.go's
// ReplyToThread, which puts domain.ValidationError.Error() straight into
// the 400 body's Message field.
func assertReplyRejected(t *testing.T, err error, wantMessage string) {
	t.Helper()

	validationErr := assert.ErrorAs[domain.ValidationError](t, err, "ReplyToThread")
	assert.Equal(t, validationErr.Error(), wantMessage, "ReplyToThread error message")
}

// assertReplyNotFound accepts either domain.ErrConversationNotFound or
// domain.ErrThreadNotFound: both surface as an HTTP 404 with no
// machine-distinguishable body, so the container driver can't recover
// which one it was any more precisely than that - see
// specifications/container/driver.go's ReplyToThread.
func assertReplyNotFound(t *testing.T, err error) {
	t.Helper()

	if !errors.Is(err, domain.ErrConversationNotFound) && !errors.Is(err, domain.ErrThreadNotFound) {
		t.Errorf("ReplyToThread returned err = %v, want domain.ErrConversationNotFound or domain.ErrThreadNotFound", err)
	}
}

// assertMessagesInOrder checks the conversation's first thread's messages -
// see assertThreadMessages, the single shared implementation every
// specification's message-list assertion funnels through.
func assertMessagesInOrder(t *testing.T, view httpConversationView, want []wantMessage) {
	t.Helper()

	assertThreadMessages(t, view.firstThread(), want)
}

// assertThreadMessages checks a specific thread's messages exactly match
// want, in order, and that every message has a real (non-zero) PostedAt -
// the one place any specification's message-list assertion is
// implemented, whether it's checking the conversation's first thread
// (assertMessagesInOrder, assertMessages) or one a specific thread further
// down the list (AddThreadSpecification, which has to assert against
// view.Threads[1] and beyond).
func assertThreadMessages(t *testing.T, thread domain.ThreadView, want []wantMessage) {
	t.Helper()

	got := make([]wantMessage, len(thread.Messages))
	for i, m := range thread.Messages {
		got[i] = wantMessage{Author: string(m.Author), Text: string(m.Text)}
	}
	assert.Equal(t, got, want, "Thread.Messages")

	for i, m := range thread.Messages {
		assert.False(t, m.PostedAt.IsZero(), "Thread.Messages[%d].PostedAt is zero, want a real timestamp", i)
	}
}

// assertThreadParticipants checks membership rather than positional
// equality - ThreadView.Participants has no guaranteed order (rule 4 of
// "thread participants") - against a specific thread, the one place any
// specification's participants assertion is implemented (assertParticipants
// delegates here for the conversation's first thread; AddThreadSpecification
// calls it directly for a thread further down the list).
func assertThreadParticipants(t *testing.T, thread domain.ThreadView, want []string) {
	t.Helper()

	assert.Len(t, thread.Participants, len(want), "Thread.Participants")
	for _, id := range want {
		assert.Contains(t, thread.Participants, domain.ParticipantID(id), "Thread.Participants")
	}
}
