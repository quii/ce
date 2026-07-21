// Package contracttest holds the shared contract-test suites for the
// event store, outbox and projection out-ports - see
// docs/adr/0009-contract-tests.md. Each suite is written against the
// out-port interface itself and run twice: once against the in-memory
// fake, once against the real Postgres adapter brought up via
// testcontainers.
package contracttest

import (
	"context"
	"testing"
	"time"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

func EventStore(t *testing.T, newStore func() out.EventStore) {
	t.Helper()

	t.Run("appending an event assigns an increasing sequence", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		first, err := store.Append(ctx, sampleEvent("conversation-1"))
		assert.NoErr(t, err, "Append")
		assert.Equal(t, first, domain.Sequence(1), "first Append's sequence - a fresh store starts from scratch")

		second, err := store.Append(ctx, sampleEvent("conversation-2"))
		assert.NoErr(t, err, "Append")
		assert.Equal(t, second, first+1, "second Append's sequence (one more than the first's sequence %d)", first)
	})

	t.Run("appending a reply after a conversation-started event assigns the next sequence", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		first, err := store.Append(ctx, sampleEvent("conversation-1"))
		assert.NoErr(t, err, "Append(ConversationStarted)")

		second, err := store.Append(ctx, sampleReplyEvent("conversation-1", "thread-1"))
		assert.NoErr(t, err, "Append(ReplyPosted)")
		assert.Equal(t, second, first+1, "Append(ReplyPosted)'s sequence (one more than the ConversationStarted's sequence %d)", first)
	})
}

// EventStoreOutbox is the combination out.EventStore and out.Outbox that
// EventStoreEnqueuesViaAppend needs, since it exercises the transactional
// interaction between the two - see docs/adr/0019-event-sourcing-transactional-outbox.md.
type EventStoreOutbox interface {
	out.EventStore
	out.Outbox
}

// EventStoreEnqueuesViaAppend proves Append and the outbox agree without
// an explicit Enqueue call: a real adapter has to enqueue the outbox row
// in the same transaction as the event to avoid the dual-write problem
// (docs/write-path.md) - a fake that only populates its "pending" state
// when Enqueue is separately called would silently disagree, exactly what
// this contract test exists to catch (docs/adr/0009-contract-tests.md).
func EventStoreEnqueuesViaAppend(t *testing.T, newStore func() EventStoreOutbox) {
	t.Helper()

	t.Run("appending an event makes it pending without a separate Enqueue call", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		seq, err := store.Append(ctx, sampleEvent("conversation-1"))
		assert.NoErr(t, err, "Append")

		pending, err := store.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 1, "Pending() after Append(seq=%d) with no Enqueue call", seq)
		assert.Equal(t, pending[0].Sequence, seq, "Pending()[0].Sequence after Append with no Enqueue call")
	})
}

func sampleEvent(conversationID string) domain.ConversationStarted {
	return domain.ConversationStarted{
		ConversationID: domain.ConversationID(conversationID),
		ThreadID:       domain.ThreadID("thread-1"),
		MessageID:      domain.MessageID("message-1"),
		Creator:        domain.PlaceholderCreator,
		ResourceURL:    domain.ResourceURL("https://example.com/orders/123"),
		ThreadTitle:    domain.ThreadTitle("Order query"),
		Author:         domain.ParticipantID("user-1"),
		Recipients:     domain.Recipients{"user-2", "user-3"},
		MessageText:    domain.MessageText("Where is my order?"),
		OccurredAt:     time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func sampleReplyEvent(conversationID, threadID string) domain.ReplyPosted {
	return domain.ReplyPosted{
		ConversationID: domain.ConversationID(conversationID),
		ThreadID:       domain.ThreadID(threadID),
		MessageID:      domain.MessageID("message-2"),
		Author:         domain.ParticipantID("user-2"),
		MessageText:    domain.MessageText("Looking into it"),
		OccurredAt:     time.Date(2024, 1, 2, 3, 5, 0, 0, time.UTC),
	}
}
