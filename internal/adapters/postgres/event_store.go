package postgres

import (
	"context"
	"fmt"

	"github.com/quii/ce/internal/domain"
)

// Append writes the event and its outbox row in one transaction - the
// transactional-outbox mechanic docs/write-path.md and
// docs/adr/0019-event-sourcing-transactional-outbox.md describe, so a
// crash between the two writes can never durably record an event the
// relay will never see.
func (s *Store) Append(ctx context.Context, event domain.ConversationStarted) (domain.Sequence, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not begin append transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	seq, err := q.InsertConversationEvent(ctx, InsertConversationEventParams{
		ConversationID: string(event.ConversationID),
		ThreadID:       string(event.ThreadID),
		MessageID:      string(event.MessageID),
		Creator:        string(event.Creator),
		ResourceUrl:    string(event.ResourceURL),
		ThreadTitle:    string(event.ThreadTitle),
		Author:         string(event.Author),
		Recipients:     recipientsToStrings(event.Recipients),
		MessageText:    string(event.MessageText),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	})
	if err != nil {
		return 0, fmt.Errorf("could not append conversation event: %w", err)
	}

	if err := q.EnqueueOutboxEntry(ctx, toEnqueueOutboxEntryParams(domain.Sequence(seq), event)); err != nil {
		return 0, fmt.Errorf("could not enqueue outbox entry for appended event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("could not commit append transaction: %w", err)
	}

	return domain.Sequence(seq), nil
}
