package in_test

import (
	"context"
	"errors"
	"testing"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

func relayEvent(conversationID string) domain.ConversationStarted {
	return domain.ConversationStarted{ConversationID: domain.ConversationID(conversationID)}
}

// TestRelay_Drain_StopsAtTheFirstGap is rule 8's concurrency guarantee:
// a caller polling after=5 must never see the projection's checkpoint
// reach or pass 5 until sequence 5 has actually been applied - even if
// sequence 6 is already visible in the outbox (conversation_events.sequence
// is assigned at INSERT time, not COMMIT time, so two concurrent Appends
// can commit out of order - see docs/adr/0019-event-sourcing-transactional-outbox.md).
func TestRelay_Drain_StopsAtTheFirstGap(t *testing.T) {
	outbox := memory.NewEventStore()
	projection := memory.NewProjection()

	ctx := context.Background()

	// Move the checkpoint to 4 through the real fake's own public API
	// (docs/adr/0008-fakes-over-mocks.md) - the "seed" conversation's
	// content is never asserted on, only the side effect on the checkpoint.
	if err := projection.Apply(ctx, relayEvent("checkpoint-seed"), 4); err != nil {
		t.Fatalf("seeding the checkpoint returned an unexpected error: %v", err)
	}

	if err := outbox.Enqueue(ctx, 6, relayEvent("conversation-6")); err != nil {
		t.Fatalf("Enqueue returned an unexpected error: %v", err)
	}

	relay := in.NewRelay(outbox, projection)
	if err := relay.Drain(ctx); err != nil {
		t.Fatalf("Drain returned an unexpected error: %v", err)
	}

	if checkpoint, err := projection.Checkpoint(ctx); err != nil || checkpoint != 4 {
		t.Errorf("Checkpoint() = (%d, %v), want (4, nil) - it must never advance past a gap", checkpoint, err)
	}

	if _, err := projection.Get(ctx, "conversation-6"); !errors.Is(err, domain.ErrConversationNotFound) {
		t.Errorf("Get(conversation-6) returned err = %v, want domain.ErrConversationNotFound - sequence 6 arrived before sequence 5, so it must not be applied yet", err)
	}

	pending, err := outbox.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending returned an unexpected error: %v", err)
	}
	if len(pending) != 1 || pending[0].Sequence != 6 {
		t.Errorf("Pending() = %#v, want sequence 6 still pending - it must not be marked done until it's actually applied", pending)
	}
}

// TestRelay_Drain_ProcessesAContiguousRunAndStopsAtTheNextGap proves Drain
// applies everything it safely can in one tick, not just "stop at the very
// first entry" - the gap check has to keep tracking forward, not give up
// after a single comparison.
func TestRelay_Drain_ProcessesAContiguousRunAndStopsAtTheNextGap(t *testing.T) {
	outbox := memory.NewEventStore()
	projection := memory.NewProjection()

	ctx := context.Background()

	if err := projection.Apply(ctx, relayEvent("checkpoint-seed"), 4); err != nil {
		t.Fatalf("seeding the checkpoint returned an unexpected error: %v", err)
	}

	toEnqueue := []struct {
		seq            domain.Sequence
		conversationID string
	}{
		{5, "conversation-5"},
		{6, "conversation-6"},
		{8, "conversation-8"},
	}
	for _, e := range toEnqueue {
		if err := outbox.Enqueue(ctx, e.seq, relayEvent(e.conversationID)); err != nil {
			t.Fatalf("Enqueue returned an unexpected error: %v", err)
		}
	}

	relay := in.NewRelay(outbox, projection)
	if err := relay.Drain(ctx); err != nil {
		t.Fatalf("Drain returned an unexpected error: %v", err)
	}

	if checkpoint, err := projection.Checkpoint(ctx); err != nil || checkpoint != 6 {
		t.Errorf("Checkpoint() = (%d, %v), want (6, nil)", checkpoint, err)
	}

	for _, conversationID := range []domain.ConversationID{"conversation-5", "conversation-6"} {
		if _, err := projection.Get(ctx, conversationID); err != nil {
			t.Errorf("Get(%s) returned err = %v, want nil - it's part of the contiguous run starting at the checkpoint", conversationID, err)
		}
	}

	if _, err := projection.Get(ctx, "conversation-8"); !errors.Is(err, domain.ErrConversationNotFound) {
		t.Errorf("Get(conversation-8) returned err = %v, want domain.ErrConversationNotFound - sequence 8 arrived before sequence 7, so it must not be applied", err)
	}

	pending, err := outbox.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending returned an unexpected error: %v", err)
	}
	if len(pending) != 1 || pending[0].Sequence != 8 {
		t.Errorf("Pending() = %#v, want only sequence 8 still pending - 5 and 6 should have been marked done", pending)
	}
}
