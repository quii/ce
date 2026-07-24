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
// enough. in.ThreadReplier is here too - the "participants unchanged after
// a reply" scenario (rule 2 of "thread participants") needs a real reply
// to post against an already-projected thread, the same shape
// ThreadReplyDriver already needs for its own specification.
type ConversationProjectionDriver interface {
	ConversationDriver
	in.ThreadReplier
	in.Relay
}

// ConversationProjectionSpecification covers rules 5-7 and 10 of the
// "start a conversation" story (now rule 1 of "thread participants" for
// rule 10's representation shape) plus rules 2 and 4 of "thread
// participants": once the relay has caught up, a read returns the full
// representation of what was written, participants are the union of
// author and recipients, and that set survives a reply unchanged.
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
		assert.Equal(t, string(view.firstThread().Title), "Order query", "Threads[0].Title")
		assertParticipants(t, view, []string{"user-1", "user-2", "user-3"})
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

		assert.Equal(t, string(view.firstThread().Title), "", "Threads[0].Title")
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

		assertParticipants(t, view, []string{"user-1"})
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

	t.Run("participants are unchanged after a reply is posted", func(t *testing.T) {
		started := startAndCatchUp(t, driver, in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{"user-2"},
			Message:     strPtr("Where is my order?"),
		})

		result, err := reply(t, driver, string(started.ID), string(started.firstThread().ID), "user-2", "Looking into it")
		assert.NoErr(t, err, "ReplyToThread")

		view := drainAndWait(t, driver, string(started.ID), result.Sequence)

		assertParticipants(t, httpConversationView{view}, []string{"user-1", "user-2"})
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

// firstThread returns the conversation's original thread - what every
// scenario predating "add a thread to a conversation" was written
// against, back when a conversation was assumed to have exactly one (rule
// 10 of "start a conversation", superseded by rule 9 of "add a thread to a
// conversation": it's still the first entry in Threads, in creation
// order).
func (v httpConversationView) firstThread() domain.ThreadView {
	return v.Threads[0]
}

// assertParticipants checks the conversation's first thread's participants
// - see assertThreadParticipants, the single shared implementation every
// specification's participants assertion funnels through.
func assertParticipants(t *testing.T, view httpConversationView, want []string) {
	t.Helper()

	assertThreadParticipants(t, view.firstThread(), want)
}

func assertMessages(t *testing.T, view httpConversationView, wantAuthor, wantText string) {
	t.Helper()

	assertThreadMessages(t, view.firstThread(), []wantMessage{{Author: wantAuthor, Text: wantText}})
}
