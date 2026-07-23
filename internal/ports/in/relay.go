package in

import (
	"context"

	"github.com/quii/ce/internal/ports/out"
)

// Relay drains the transactional outbox into the read-side projection -
// see docs/write-path.md. Drain is a plain, synchronous operation: in
// production it's called repeatedly by a trivial loop-with-ticker at the
// composition root (cmd/relay), which has no business logic worth testing
// itself. Drain is what's actually under test.
type Relay interface {
	Drain(ctx context.Context) error
}

type relayUseCase struct {
	outbox     out.Outbox
	projection out.Projection
}

func NewRelay(outbox out.Outbox, projection out.Projection) Relay {
	return &relayUseCase{outbox: outbox, projection: projection}
}

// Drain only ever advances the checkpoint contiguously. A Postgres
// sequence column is assigned at INSERT time, not COMMIT time, so two
// concurrent Append calls can commit out of order - Pending can return a
// later sequence before an earlier one has committed at all. Stopping at
// the first gap between the projection's checkpoint and what Pending
// returns is what stops the checkpoint from ever advancing past data
// that was never actually applied - see rule 8 of the "start a
// conversation" story and docs/adr/0019-event-sourcing-transactional-outbox.md.
//
// The whole contiguous run is applied in one Projection.Apply call, not
// one call per entry: a single StartConversation write lands as several
// contiguous outbox rows (docs/adr/0029-fine-grained-events.md), and
// applying them individually would let a reader observe e.g. a
// ThreadStarted with no MessagePosted companion yet, in between two
// separate Drain steps - exactly the partially-applied state
// Projection.Apply's batching exists to rule out.
func (r *relayUseCase) Drain(ctx context.Context) error {
	pending, err := r.outbox.Pending(ctx)
	if err != nil {
		return err
	}

	checkpoint, err := r.projection.Checkpoint(ctx)
	if err != nil {
		return err
	}
	next := checkpoint + 1

	var run []out.OutboxEntry
	for _, entry := range pending {
		if entry.Sequence != next {
			break
		}
		run = append(run, entry)
		next++
	}

	if len(run) == 0 {
		return nil
	}

	if err := r.projection.Apply(ctx, run...); err != nil {
		return err
	}

	for _, entry := range run {
		if err := r.outbox.MarkDone(ctx, entry.Sequence); err != nil {
			return err
		}
	}

	return nil
}
