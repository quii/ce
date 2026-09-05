package in

import (
	"context"

	"github.com/quii/ce/internal/ports/out"
)

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
