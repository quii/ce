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

		first, err := store.Append(ctx, sampleConversationCreated("conversation-1"))
		assert.NoErr(t, err, "Append")
		assert.Equal(t, first, domain.Sequence(1), "first Append's sequence - a fresh store starts from scratch")

		second, err := store.Append(ctx, sampleConversationCreated("conversation-2"))
		assert.NoErr(t, err, "Append")
		assert.Equal(t, second, first+1, "second Append's sequence (one more than the first's sequence %d)", first)
	})

	t.Run("appending a message after a conversation-created event assigns the next sequence", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		first, err := store.Append(ctx, sampleConversationCreated("conversation-1"))
		assert.NoErr(t, err, "Append(ConversationCreated)")

		second, err := store.Append(ctx, sampleMessagePosted("conversation-1", "thread-1", "message-1"))
		assert.NoErr(t, err, "Append(MessagePosted)")
		assert.Equal(t, second, first+1, "Append(MessagePosted)'s sequence (one more than the ConversationCreated's sequence %d)", first)
	})

	t.Run("appending a batch of events returns the sequence of the last event in the batch", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		first, err := store.Append(ctx, sampleConversationCreated("conversation-1"))
		assert.NoErr(t, err, "Append")

		last, err := store.Append(ctx,
			sampleThreadStarted("conversation-1", "thread-1"),
			sampleMessagePosted("conversation-1", "thread-1", "message-1"),
		)
		assert.NoErr(t, err, "Append(batch of two events)")
		assert.Equal(t, last, first+2, "Append(batch)'s sequence (two more than the first Append's sequence %d, since the batch has two events)", first)
	})

	t.Run("listing events for a conversation returns them in append order", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		threadStarted := sampleThreadStarted("conversation-1", "thread-1")
		messagePosted := sampleMessagePosted("conversation-1", "thread-1", "message-1")

		_, err := store.Append(ctx, created, threadStarted, messagePosted)
		assert.NoErr(t, err, "Append(batch of three events)")

		records, err := store.ListByConversation(ctx, domain.ConversationID("conversation-1"))
		assert.NoErr(t, err, "ListByConversation")
		assert.Len(t, records, 3, "ListByConversation")

		wantSequences := []domain.Sequence{1, 2, 3}
		gotSequences := make([]domain.Sequence, len(records))
		for i, record := range records {
			gotSequences[i] = record.Sequence
		}
		assert.Equal(t, gotSequences, wantSequences, "ListByConversation sequence order")

		_, ok := records[0].Event.(domain.ConversationCreated)
		assert.True(t, ok, "ListByConversation()[0].Event = %#v, want a domain.ConversationCreated", records[0].Event)
		_, ok = records[1].Event.(domain.ThreadStarted)
		assert.True(t, ok, "ListByConversation()[1].Event = %#v, want a domain.ThreadStarted", records[1].Event)
		_, ok = records[2].Event.(domain.MessagePosted)
		assert.True(t, ok, "ListByConversation()[2].Event = %#v, want a domain.MessagePosted", records[2].Event)
	})

	t.Run("listing events only returns events belonging to the requested conversation", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		_, err := store.Append(ctx, sampleConversationCreated("conversation-1"))
		assert.NoErr(t, err, "Append(conversation-1)")
		_, err = store.Append(ctx, sampleConversationCreated("conversation-2"))
		assert.NoErr(t, err, "Append(conversation-2)")

		records, err := store.ListByConversation(ctx, domain.ConversationID("conversation-2"))
		assert.NoErr(t, err, "ListByConversation")
		assert.Len(t, records, 1, "ListByConversation(conversation-2)")

		got, ok := records[0].Event.(domain.ConversationCreated)
		assert.True(t, ok, "ListByConversation(conversation-2)[0].Event = %#v, want a domain.ConversationCreated", records[0].Event)
		assert.Equal(t, got.ConversationID, domain.ConversationID("conversation-2"), "ListByConversation(conversation-2)[0].Event.ConversationID")
	})

	t.Run("listing events for a conversation that has never had an event appended returns an empty list", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		records, err := store.ListByConversation(ctx, domain.ConversationID("missing-conversation"))
		assert.NoErr(t, err, "ListByConversation")
		assert.Len(t, records, 0, "ListByConversation(missing-conversation)")
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

		seq, err := store.Append(ctx, sampleConversationCreated("conversation-1"))
		assert.NoErr(t, err, "Append")

		pending, err := store.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 1, "Pending() after Append(seq=%d) with no Enqueue call", seq)
		assert.Equal(t, pending[0].Sequence, seq, "Pending()[0].Sequence after Append with no Enqueue call")
	})

	// TestStartConversation_RaisesThreeEventsAtomically
	// (internal/ports/in/start_conversation_test.go) proves this same
	// behaviour end-to-end through the real use case; this subtest proves
	// it directly at the out-port level, against both the fake and the
	// real Postgres adapter - see docs/adr/0029-fine-grained-events.md.
	t.Run("appending a batch of events assigns sequential sequences and enqueues all of them, in order, with one call", func(t *testing.T) {
		store := newStore()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		threadStarted := sampleThreadStarted("conversation-1", "thread-1")
		messagePosted := sampleMessagePosted("conversation-1", "thread-1", "message-1")

		last, err := store.Append(ctx, created, threadStarted, messagePosted)
		assert.NoErr(t, err, "Append(batch of three events)")
		assert.Equal(t, last, domain.Sequence(3), "Append(batch)'s returned sequence - the last of the three")

		pending, err := store.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 3, "Pending() after Append(batch) with no Enqueue call")

		wantSequences := []domain.Sequence{1, 2, 3}
		gotSequences := make([]domain.Sequence, len(pending))
		for i, entry := range pending {
			gotSequences[i] = entry.Sequence
		}
		assert.Equal(t, gotSequences, wantSequences, "Pending() sequence order after Append(batch)")

		if _, ok := pending[0].Event.(domain.ConversationCreated); !ok {
			t.Fatalf("Pending()[0].Event = %#v, want a domain.ConversationCreated", pending[0].Event)
		}
		if _, ok := pending[1].Event.(domain.ThreadStarted); !ok {
			t.Fatalf("Pending()[1].Event = %#v, want a domain.ThreadStarted", pending[1].Event)
		}
		if _, ok := pending[2].Event.(domain.MessagePosted); !ok {
			t.Fatalf("Pending()[2].Event = %#v, want a domain.MessagePosted", pending[2].Event)
		}
	})
}

func sampleConversationCreated(conversationID string) domain.ConversationCreated {
	return domain.ConversationCreated{
		ConversationID: domain.ConversationID(conversationID),
		Creator:        domain.PlaceholderCreator,
		ResourceURL:    domain.ResourceURL("https://example.com/orders/123"),
		OccurredAt:     time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func sampleThreadStarted(conversationID, threadID string) domain.ThreadStarted {
	return domain.ThreadStarted{
		ConversationID: domain.ConversationID(conversationID),
		ThreadID:       domain.ThreadID(threadID),
		ThreadTitle:    domain.ThreadTitle("Order query"),
		Author:         domain.ParticipantID("user-1"),
		Recipients:     domain.Recipients{"user-2", "user-3"},
		OccurredAt:     time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func sampleMessagePosted(conversationID, threadID, messageID string) domain.MessagePosted {
	return domain.MessagePosted{
		ConversationID: domain.ConversationID(conversationID),
		ThreadID:       domain.ThreadID(threadID),
		MessageID:      domain.MessageID(messageID),
		Author:         domain.ParticipantID("user-1"),
		MessageText:    domain.MessageText("Where is my order?"),
		OccurredAt:     time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}
