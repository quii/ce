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

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

func EventStore(t *testing.T, newStore func() out.EventStore) {
	t.Helper()

	t.Run("appending an event assigns an increasing sequence", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		first, err := store.Append(ctx, sampleEvent("conversation-1"))
		if err != nil {
			t.Fatalf("Append returned an unexpected error: %v", err)
		}
		if first != 1 {
			t.Errorf("first Append's sequence = %d, want 1 - a fresh store starts from scratch", first)
		}

		second, err := store.Append(ctx, sampleEvent("conversation-2"))
		if err != nil {
			t.Fatalf("Append returned an unexpected error: %v", err)
		}
		if second != first+1 {
			t.Errorf("second Append's sequence = %d, want %d (one more than the first's sequence %d)", second, first+1, first)
		}
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
		if err != nil {
			t.Fatalf("Append returned an unexpected error: %v", err)
		}

		pending, err := store.Pending(ctx)
		if err != nil {
			t.Fatalf("Pending returned an unexpected error: %v", err)
		}
		if len(pending) != 1 || pending[0].Sequence != seq {
			t.Errorf("Pending() after Append(seq=%d) with no Enqueue call = %#v, want exactly that entry pending", seq, pending)
		}
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
