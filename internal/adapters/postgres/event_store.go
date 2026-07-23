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
	switch e := event.(type) {
	case domain.ConversationCreated:
		seq, err := q.InsertConversationCreatedEvent(ctx, toInsertConversationCreatedEventParams(e))
		if err != nil {
			return 0, fmt.Errorf("could not append conversation created event: %w", err)
		}
		if err := q.EnqueueConversationCreatedOutboxEntry(ctx, toEnqueueConversationCreatedOutboxEntryParams(domain.Sequence(seq), e)); err != nil {
			return 0, fmt.Errorf("could not enqueue outbox entry for appended event: %w", err)
		}
		return domain.Sequence(seq), nil
	case domain.ThreadStarted:
		seq, err := q.InsertThreadStartedEvent(ctx, toInsertThreadStartedEventParams(e))
		if err != nil {
			return 0, fmt.Errorf("could not append thread started event: %w", err)
		}
		if err := q.EnqueueThreadStartedOutboxEntry(ctx, toEnqueueThreadStartedOutboxEntryParams(domain.Sequence(seq), e)); err != nil {
			return 0, fmt.Errorf("could not enqueue outbox entry for appended event: %w", err)
		}
		return domain.Sequence(seq), nil
	case domain.MessagePosted:
		seq, err := q.InsertMessagePostedEvent(ctx, toInsertMessagePostedEventParams(e))
		if err != nil {
			return 0, fmt.Errorf("could not append message posted event: %w", err)
		}
		if err := q.EnqueueMessagePostedOutboxEntry(ctx, toEnqueueMessagePostedOutboxEntryParams(domain.Sequence(seq), e)); err != nil {
			return 0, fmt.Errorf("could not enqueue outbox entry for appended event: %w", err)
		}
		return domain.Sequence(seq), nil
	default:
		return 0, fmt.Errorf("cannot append event of unrecognized type %T", event)
	}
}
