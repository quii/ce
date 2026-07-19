package contracttest

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

func Outbox(t *testing.T, newOutbox func() out.Outbox) {
	t.Helper()

	t.Run("an enqueued entry is pending until marked done", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()
		event := sampleEvent("conversation-1")

		if err := outbox.Enqueue(ctx, 1, event); err != nil {
			t.Fatalf("Enqueue returned an unexpected error: %v", err)
		}

		pending, err := outbox.Pending(ctx)
		if err != nil {
			t.Fatalf("Pending returned an unexpected error: %v", err)
		}
		if len(pending) != 1 {
			t.Fatalf("Pending() = %#v, want exactly one entry", pending)
		}
		if pending[0].Sequence != 1 {
			t.Errorf("Pending()[0].Sequence = %d, want 1", pending[0].Sequence)
		}
		if pending[0].Event.ConversationID != event.ConversationID {
			t.Errorf("Pending()[0].Event.ConversationID = %q, want %q", pending[0].Event.ConversationID, event.ConversationID)
		}

		if err := outbox.MarkDone(ctx, 1); err != nil {
			t.Fatalf("MarkDone returned an unexpected error: %v", err)
		}

		pending, err = outbox.Pending(ctx)
		if err != nil {
			t.Fatalf("Pending returned an unexpected error: %v", err)
		}
		if len(pending) != 0 {
			t.Errorf("Pending() after MarkDone = %#v, want none", pending)
		}
	})

	t.Run("pending entries are returned in sequence order", func(t *testing.T) {
		outbox := newOutbox()
		ctx := context.Background()

		if err := outbox.Enqueue(ctx, 2, sampleEvent("conversation-2")); err != nil {
			t.Fatalf("Enqueue returned an unexpected error: %v", err)
		}
		if err := outbox.Enqueue(ctx, 1, sampleEvent("conversation-1")); err != nil {
			t.Fatalf("Enqueue returned an unexpected error: %v", err)
		}

		pending, err := outbox.Pending(ctx)
		if err != nil {
			t.Fatalf("Pending returned an unexpected error: %v", err)
		}

		want := []domain.Sequence{1, 2}
		if len(pending) != len(want) {
			t.Fatalf("Pending() = %#v, want %d entries", pending, len(want))
		}
		for i, seq := range want {
			if pending[i].Sequence != seq {
				t.Errorf("Pending()[%d].Sequence = %d, want %d", i, pending[i].Sequence, seq)
			}
		}
	})
}
