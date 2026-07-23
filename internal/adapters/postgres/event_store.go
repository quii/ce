package postgres

import (
	"context"
	"fmt"

	"github.com/quii/ce/internal/domain"
)

// Append writes every event in the batch, plus its outbox row, in one
// transaction - the transactional-outbox mechanic docs/write-path.md and
// docs/adr/0019-event-sourcing-transactional-outbox.md describe, so a
// crash partway through a multi-event write can never durably record a
// partial batch (docs/adr/0029-fine-grained-events.md). It returns the
// sequence of the last event appended.
func (s *Store) Append(ctx context.Context, events ...domain.Event) (domain.Sequence, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not begin append transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	var seq domain.Sequence
	for _, event := range events {
		seq, err = appendEvent(ctx, q, event)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("could not commit append transaction: %w", err)
	}

	return seq, nil
}

func appendEvent(ctx context.Context, q *Queries, event domain.Event) (domain.Sequence, error) {
	row, err := marshalEvent(event)
	if err != nil {
		return 0, fmt.Errorf("could not append event: %w", err)
	}

	seq, err := q.InsertEvent(ctx, InsertEventParams{
		EventType:      row.EventType,
		ConversationID: string(row.ConversationID),
		OccurredAt:     toTimestamptz(row.OccurredAt),
		Payload:        row.Payload,
	})
	if err != nil {
		return 0, fmt.Errorf("could not append %s event: %w", row.EventType, err)
	}

	if err := q.EnqueueOutboxEntry(ctx, EnqueueOutboxEntryParams{
		Sequence:       seq,
		EventType:      row.EventType,
		ConversationID: string(row.ConversationID),
		OccurredAt:     toTimestamptz(row.OccurredAt),
		Payload:        row.Payload,
	}); err != nil {
		return 0, fmt.Errorf("could not enqueue outbox entry for appended event: %w", err)
	}

	return domain.Sequence(seq), nil
}
