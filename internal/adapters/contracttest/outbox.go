package contracttest

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

func Outbox(t *testing.T, newOutbox func() out.Outbox) {
	t.Helper()

	t.Run("an enqueued entry is pending until marked done", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()
		event := sampleConversationCreated("conversation-1")

		assert.NoErr(t, outbox.Enqueue(ctx, 1, event), "Enqueue")

		pending, err := outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 1, "Pending()")
		assert.Equal(t, pending[0].Sequence, domain.Sequence(1), "Pending()[0].Sequence")
		got, ok := pending[0].Event.(domain.ConversationCreated)
		if !ok {
			t.Fatalf("Pending()[0].Event = %#v, want a domain.ConversationCreated", pending[0].Event)
		}
		assert.Equal(t, got.ConversationID, event.ConversationID, "Pending()[0].Event.(domain.ConversationCreated).ConversationID")

		assert.NoErr(t, outbox.MarkDone(ctx, 1), "MarkDone")

		pending, err = outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 0, "Pending() after MarkDone")
	})

	t.Run("an enqueued message round-trips through Pending", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()
		event := sampleMessagePosted("conversation-1", "thread-1", "message-1")

		assert.NoErr(t, outbox.Enqueue(ctx, 1, event), "Enqueue")

		pending, err := outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 1, "Pending()")
		got, ok := pending[0].Event.(domain.MessagePosted)
		if !ok {
			t.Fatalf("Pending()[0].Event = %#v, want a domain.MessagePosted", pending[0].Event)
		}
		assert.Equal(t, got.ConversationID, event.ConversationID, "Pending()[0].Event.(domain.MessagePosted).ConversationID")
		assert.Equal(t, got.Author, event.Author, "Pending()[0].Event.(domain.MessagePosted).Author")
	})

	t.Run("an enqueued thread-started event round-trips through Pending", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()
		event := sampleThreadStarted("conversation-1", "thread-1")

		assert.NoErr(t, outbox.Enqueue(ctx, 1, event), "Enqueue")

		pending, err := outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 1, "Pending()")
		got, ok := pending[0].Event.(domain.ThreadStarted)
		if !ok {
			t.Fatalf("Pending()[0].Event = %#v, want a domain.ThreadStarted", pending[0].Event)
		}
		assert.Equal(t, got.ConversationID, event.ConversationID, "Pending()[0].Event.(domain.ThreadStarted).ConversationID")
		assert.Equal(t, got.ThreadID, event.ThreadID, "Pending()[0].Event.(domain.ThreadStarted).ThreadID")
	})

	t.Run("pending entries are returned in sequence order", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()

		assert.NoErr(t, outbox.Enqueue(ctx, 2, sampleConversationCreated("conversation-2")), "Enqueue")
		assert.NoErr(t, outbox.Enqueue(ctx, 1, sampleConversationCreated("conversation-1")), "Enqueue")

		pending, err := outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")

		want := []domain.Sequence{1, 2}
		assert.Len(t, pending, len(want), "Pending()")
		got := make([]domain.Sequence, len(pending))
		for i, entry := range pending {
			got[i] = entry.Sequence
		}
		assert.Equal(t, got, want, "Pending() sequence order")
	})
}
