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

// ListByConversation reads straight from conversation_events, ordered by
// sequence (append order) - see docs/adr/0026-sql-spec-first-with-sqlc.md
// and "list a conversation's events" rule 3. An empty result means no
// event has ever been appended for id, which the caller (in.EventLister)
// treats as "conversation not found" - rule 2.
func (s *Store) ListByConversation(ctx context.Context, id domain.ConversationID) ([]domain.EventRecord, error) {
	rows, err := s.queries.ListConversationEvents(ctx, string(id))
	if err != nil {
		return nil, fmt.Errorf("could not list events for conversation %s: %w", id, err)
	}

	records := make([]domain.EventRecord, len(rows))
	for i, row := range rows {
		event, err := toDomainEvent(row.Sequence, row.EventType, row.ConversationID, row.OccurredAt, row.Payload)
		if err != nil {
			return nil, err
		}
		records[i] = domain.EventRecord{Sequence: domain.Sequence(row.Sequence), Event: event}
	}

	return records, nil
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
