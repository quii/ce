package postgres

import (
	"context"
	"fmt"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// Enqueue is idempotent (EnqueueOutboxEntry is an upsert-free insert with
// ON CONFLICT (sequence) DO NOTHING): Append already enqueues the outbox
// row itself, in the same transaction as the event, so the use case's
// subsequent Enqueue call is always a safe no-op in production - this
// method only does real work when a caller (a contract test, say) drives
// Outbox in isolation, without a prior Append.
func (s *Store) Enqueue(ctx context.Context, seq domain.Sequence, event domain.ConversationStarted) error {
	if err := s.queries.EnqueueOutboxEntry(ctx, toEnqueueOutboxEntryParams(seq, event)); err != nil {
		return fmt.Errorf("could not enqueue outbox entry: %w", err)
	}

	return nil
}

func (s *Store) Pending(ctx context.Context) ([]out.OutboxEntry, error) {
	rows, err := s.queries.ListPendingOutboxEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list pending outbox entries: %w", err)
	}

	entries := make([]out.OutboxEntry, len(rows))
	for i, row := range rows {
		entries[i] = out.OutboxEntry{
			Sequence: domain.Sequence(row.Sequence),
			Event: domain.ConversationStarted{
				ConversationID: domain.ConversationID(row.ConversationID),
				ThreadID:       domain.ThreadID(row.ThreadID),
				MessageID:      domain.MessageID(row.MessageID),
				Creator:        domain.CreatorID(row.Creator),
				ResourceURL:    domain.ResourceURL(row.ResourceUrl),
				ThreadTitle:    domain.ThreadTitle(row.ThreadTitle),
				Author:         domain.ParticipantID(row.Author),
				Recipients:     stringsToRecipients(row.Recipients),
				MessageText:    domain.MessageText(row.MessageText),
				OccurredAt:     row.OccurredAt.Time,
			},
		}
	}

	return entries, nil
}

func (s *Store) MarkDone(ctx context.Context, seq domain.Sequence) error {
	if err := s.queries.MarkOutboxEntryDone(ctx, int64(seq)); err != nil {
		return fmt.Errorf("could not mark outbox entry %d done: %w", seq, err)
	}

	return nil
}
