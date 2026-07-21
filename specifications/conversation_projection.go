package specifications

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// ConversationProjectionDriver additionally exposes Relay.Drain, so a
// specification can ask a driver to make sure a write has been processed
// - see docs/adr/0022-specifications-and-drivers.md. The in-process
// driver's Drain is the real relay, called synchronously: by the time it
// returns, the very next read is guaranteed to be caught up. The
// container driver's Drain is a no-op - it talks to a real, separate
// relay container draining on its own ticker, with no HTTP surface to
// trigger it on demand - so waitForProjection below is what actually
// bridges that gap, polling the read exactly the way any HTTP client is
// expected to (docs/write-path.md) rather than assuming Drain alone was
// enough.
type ConversationProjectionDriver interface {
	ConversationDriver
	in.Relay
}

// ConversationProjectionSpecification covers rules 5-7 and 10 of the
// "start a conversation" story: once the relay has caught up, a read
// returns the full representation of what was written.
func ConversationProjectionSpecification(t *testing.T, driver ConversationProjectionDriver) {
	t.Helper()

	t.Run("a conversation is started with all required fields", func(t *testing.T) {
		view := startAndCatchUp(t, driver, in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{"user-2", "user-3"},
			Message:     strPtr("Where is my order?"),
		})

		assert.Equal(t, string(view.ResourceURL), "https://example.com/orders/123", "ResourceURL")
		assert.Equal(t, string(view.Thread.Title), "Order query", "Thread.Title")
		assertRecipients(t, view, []string{"user-2", "user-3"})
		assertMessages(t, view, "user-1", "Where is my order?")
	})

	t.Run("empty string values are accepted", func(t *testing.T) {
		view := startAndCatchUp(t, driver, in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr(""),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{},
			Message:     strPtr(""),
		})

		assert.Equal(t, string(view.Thread.Title), "", "Thread.Title")
		assertMessages(t, view, "user-1", "")
	})

	t.Run("empty recipients are accepted", func(t *testing.T) {
		view := startAndCatchUp(t, driver, in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{},
			Message:     strPtr("Where is my order?"),
		})

		assertRecipients(t, view, []string{})
	})

	t.Run("reading a conversation after the projection has caught up returns it", func(t *testing.T) {
		view := startAndCatchUp(t, driver, in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{"user-2"},
			Message:     strPtr("Where is my order?"),
		})

		assert.Equal(t, string(view.ResourceURL), "https://example.com/orders/123", "ResourceURL")
	})
}

func startAndCatchUp(t *testing.T, driver ConversationProjectionDriver, cmd in.StartConversationCommand) httpConversationView {
	t.Helper()
	ctx := context.Background()

	started, err := driver.StartConversation(ctx, cmd)
	assert.NoErr(t, err, "StartConversation(%+v)", cmd)

	assert.NoErr(t, driver.Drain(ctx), "Drain")

	view := waitForProjection(t, driver, string(started.ConversationID), int64(started.Sequence))

	return httpConversationView{view}
}

// waitForProjection polls GetConversation until the projection has caught
// up to after, or the deadline elapses. For a driver whose Drain already
// did the work synchronously, the very first attempt succeeds - this adds
// no latency there. For a driver backed by a real, independently-ticking
// relay, this is the only thing that makes "eventually resolves" an
// observable, non-flaky outcome (a ticker-driven poll, never time.Sleep -
// docs/adr/0021-no-flaky-tests.md).
func waitForProjection(t *testing.T, driver ConversationDriver, conversationID string, after int64) domain.ConversationView {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		view, err := driver.GetConversation(ctx, in.GetConversationCommand{
			ConversationID: conversationID,
			After:          &after,
		})
		if err == nil {
			return view
		}
		if !errors.Is(err, domain.ErrProjectionNotCaughtUp) {
			t.Fatalf("GetConversation(after=%d) returned an unexpected error: %v", after, err)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("projection did not catch up to sequence %d before the test timeout", after)
		case <-ticker.C:
		}
	}
}

type httpConversationView struct {
	domain.ConversationView
}

// assertRecipients checks membership rather than positional equality -
// domain.Recipients is a set (docs/domain rule: duplicates are rejected
// at construction, docs/adr - internal/domain/conversation.go's
// NewRecipients), so two recipient lists with the same members in a
// different order are the same value as far as this story's rules go.
func assertRecipients(t *testing.T, view httpConversationView, want []string) {
	t.Helper()

	assert.Len(t, view.Thread.Recipients, len(want), "Thread.Recipients")
	for _, id := range want {
		assert.Contains(t, view.Thread.Recipients, domain.ParticipantID(id), "Thread.Recipients")
	}
}

func assertMessages(t *testing.T, view httpConversationView, wantAuthor, wantText string) {
	t.Helper()

	assert.Len(t, view.Thread.Messages, 1, "Thread.Messages")

	got := view.Thread.Messages[0]
	assert.Equal(t, string(got.Author), wantAuthor, "Messages[0].Author")
	assert.Equal(t, string(got.Text), wantText, "Messages[0].Text")
	assert.False(t, got.PostedAt.IsZero(), "Messages[0].PostedAt is zero, want a real timestamp")
}
