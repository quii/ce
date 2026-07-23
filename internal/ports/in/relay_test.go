package in_test

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/internal/ports/out"
)

func relayEvent(conversationID string) domain.ConversationCreated {
	return domain.ConversationCreated{ConversationID: domain.ConversationID(conversationID)}
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
	assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 4, Event: relayEvent("checkpoint-seed")}), "seeding the checkpoint")
	assert.NoErr(t, outbox.Enqueue(ctx, 6, relayEvent("conversation-6")), "Enqueue")

	relay := in.NewRelay(outbox, projection)
	assert.NoErr(t, relay.Drain(ctx), "Drain")

	checkpoint, err := projection.Checkpoint(ctx)
	assert.NoErr(t, err, "Checkpoint")
	assert.Equal(t, checkpoint, domain.Sequence(4), "Checkpoint() - it must never advance past a gap")

	_, err = projection.Get(ctx, "conversation-6")
	assert.ErrorIs(t, err, domain.ErrConversationNotFound, "Get(conversation-6) - sequence 6 arrived before sequence 5, so it must not be applied yet")

	pending, err := outbox.Pending(ctx)
	assert.NoErr(t, err, "Pending")
	assert.Len(t, pending, 1, "Pending() (still pending)")
	assert.Equal(t, pending[0].Sequence, domain.Sequence(6), "Pending()[0].Sequence - it must not be marked done until it's actually applied")
}

// TestRelay_Drain_ProcessesAContiguousRunAndStopsAtTheNextGap proves Drain
// applies everything it safely can in one tick, not just "stop at the very
// first entry" - the gap check has to keep tracking forward, not give up
// after a single comparison.
func TestRelay_Drain_ProcessesAContiguousRunAndStopsAtTheNextGap(t *testing.T) {
	outbox := memory.NewEventStore()
	projection := memory.NewProjection()

	ctx := context.Background()

	assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 4, Event: relayEvent("checkpoint-seed")}), "seeding the checkpoint")

	toEnqueue := []struct {
		seq            domain.Sequence
		conversationID string
	}{
		{5, "conversation-5"},
		{6, "conversation-6"},
		{8, "conversation-8"},
	}
	for _, e := range toEnqueue {
		assert.NoErr(t, outbox.Enqueue(ctx, e.seq, relayEvent(e.conversationID)), "Enqueue")
	}

	relay := in.NewRelay(outbox, projection)
	assert.NoErr(t, relay.Drain(ctx), "Drain")

	checkpoint, err := projection.Checkpoint(ctx)
	assert.NoErr(t, err, "Checkpoint")
	assert.Equal(t, checkpoint, domain.Sequence(6), "Checkpoint() - it must have advanced across both 5 and 6, not just the first of the run")

	// Not asserting Get(conversation-5)/Get(conversation-6) succeed here: a
	// bare ConversationCreated on its own never makes a conversation
	// "found" (docs/adr/0029-fine-grained-events.md - a real ThreadStarted
	// always accompanies it), so - as with the seed above - relayEvent's
	// content is never asserted on. Checkpoint() and Pending() below are
	// what actually prove 5 and 6 were applied and 8 wasn't.
	_, err = projection.Get(ctx, "conversation-8")
	assert.ErrorIs(t, err, domain.ErrConversationNotFound, "Get(conversation-8) - sequence 8 arrived before sequence 7, so it must not be applied")

	pending, err := outbox.Pending(ctx)
	assert.NoErr(t, err, "Pending")
	assert.Len(t, pending, 1, "Pending() (still pending - 5 and 6 should have been marked done)")
	assert.Equal(t, pending[0].Sequence, domain.Sequence(8), "Pending()[0].Sequence")
}

// batchSizeRecordingProjection wraps a real memory.Projection, driving it
// through its own public Apply as usual (docs/adr/0008-fakes-over-mocks.md),
// but also recording how many entries each call received - the only way to
// observe, from outside Drain, whether it grouped a contiguous run into one
// Projection.Apply call or made one call per entry.
type batchSizeRecordingProjection struct {
	*memory.Projection
	applyBatchSizes []int
}

func (p *batchSizeRecordingProjection) Apply(ctx context.Context, entries ...out.OutboxEntry) error {
	p.applyBatchSizes = append(p.applyBatchSizes, len(entries))
	return p.Projection.Apply(ctx, entries...)
}

// TestRelay_Drain_AppliesAContiguousRunInOneApplyCall proves a gap found
// during review of the "conversation event split" story: a single
// StartConversation write raises three events (ConversationCreated,
// ThreadStarted, MessagePosted) as three contiguous outbox rows - applying
// them one at a time would let a plain, unconditional GET observe e.g. a
// thread with no messages yet in the gap between two separate Drain steps,
// exactly the partially-applied state Projection.Apply's own batching
// exists to rule out (docs/adr/0029-fine-grained-events.md).
func TestRelay_Drain_AppliesAContiguousRunInOneApplyCall(t *testing.T) {
	outbox := memory.NewEventStore()
	projection := &batchSizeRecordingProjection{Projection: memory.NewProjection()}

	ctx := context.Background()

	assert.NoErr(t, outbox.Enqueue(ctx, 1, relayEvent("conversation-1")), "Enqueue")
	assert.NoErr(t, outbox.Enqueue(ctx, 2, relayEvent("conversation-2")), "Enqueue")
	assert.NoErr(t, outbox.Enqueue(ctx, 3, relayEvent("conversation-3")), "Enqueue")

	relay := in.NewRelay(outbox, projection)
	assert.NoErr(t, relay.Drain(ctx), "Drain")

	assert.Equal(t, projection.applyBatchSizes, []int{3}, "Projection.Apply's batch sizes - the whole contiguous run in one call, not one call per entry")
}
