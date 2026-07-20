package specifications

import (
	"context"
	"errors"
	"testing"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

func startThreadAndCatchUp(t *testing.T, driver ThreadReplyDriver, author string, recipients []string, message string) (conversationID, threadID string) {
	t.Helper()
	ctx := context.Background()

	started, err := driver.StartConversation(ctx, in.StartConversationCommand{
		ResourceURL: strPtr("https://example.com/orders/123"),
		ThreadTitle: strPtr("Order query"),
		Author:      strPtr(author),
		Recipients:  &recipients,
		Message:     strPtr(message),
	})
	if err != nil {
		t.Fatalf("StartConversation returned an unexpected error: %v", err)
	}

	view := drainAndWait(t, driver, string(started.ConversationID), started.Sequence)

	return string(started.ConversationID), string(view.Thread.ID)
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

func drainAndWait(t *testing.T, driver ThreadReplyDriver, conversationID string, seq domain.Sequence) domain.ConversationView {
	t.Helper()
	ctx := context.Background()

	if err := driver.Drain(ctx); err != nil {
		t.Fatalf("Drain returned an unexpected error: %v", err)
	}

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

	var validationErr domain.ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("ReplyToThread returned err = %v, want a domain.ValidationError", err)
		return
	}
	if validationErr.Error() != wantMessage {
		t.Errorf("ReplyToThread returned err = %q, want message %q", validationErr.Error(), wantMessage)
	}
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

func assertMessagesInOrder(t *testing.T, view httpConversationView, want []wantMessage) {
	t.Helper()

	if len(view.Thread.Messages) != len(want) {
		t.Fatalf("Thread.Messages = %#v, want %d messages in order: %#v", view.Thread.Messages, len(want), want)
	}

	for i, w := range want {
		got := view.Thread.Messages[i]
		if string(got.Author) != w.author {
			t.Errorf("Messages[%d].Author = %q, want %q", i, got.Author, w.author)
		}
		if string(got.Text) != w.text {
			t.Errorf("Messages[%d].Text = %q, want %q", i, got.Text, w.text)
		}
		if got.PostedAt.IsZero() {
			t.Errorf("Messages[%d].PostedAt is zero, want a real timestamp", i)
		}
	}
}
