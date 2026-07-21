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
		event := sampleEvent("conversation-1")

		assert.NoErr(t, outbox.Enqueue(ctx, 1, event), "Enqueue")

		pending, err := outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 1, "Pending()")
		assert.Equal(t, pending[0].Sequence, domain.Sequence(1), "Pending()[0].Sequence")
		got, ok := pending[0].Event.(domain.ConversationStarted)
		if !ok {
			t.Fatalf("Pending()[0].Event = %#v, want a domain.ConversationStarted", pending[0].Event)
		}
		assert.Equal(t, got.ConversationID, event.ConversationID, "Pending()[0].Event.(domain.ConversationStarted).ConversationID")

		assert.NoErr(t, outbox.MarkDone(ctx, 1), "MarkDone")

		pending, err = outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 0, "Pending() after MarkDone")
	})

	t.Run("an enqueued reply round-trips through Pending", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()
		event := sampleReplyEvent("conversation-1", "thread-1")

		assert.NoErr(t, outbox.Enqueue(ctx, 1, event), "Enqueue")

		pending, err := outbox.Pending(ctx)
		assert.NoErr(t, err, "Pending")
		assert.Len(t, pending, 1, "Pending()")
		got, ok := pending[0].Event.(domain.ReplyPosted)
		if !ok {
			t.Fatalf("Pending()[0].Event = %#v, want a domain.ReplyPosted", pending[0].Event)
		}
		assert.Equal(t, got.ConversationID, event.ConversationID, "Pending()[0].Event.(domain.ReplyPosted).ConversationID")
		assert.Equal(t, got.Author, event.Author, "Pending()[0].Event.(domain.ReplyPosted).Author")
	})

	t.Run("pending entries are returned in sequence order", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()

		assert.NoErr(t, outbox.Enqueue(ctx, 2, sampleEvent("conversation-2")), "Enqueue")
		assert.NoErr(t, outbox.Enqueue(ctx, 1, sampleEvent("conversation-1")), "Enqueue")

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
