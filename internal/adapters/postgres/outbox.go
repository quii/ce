package postgres

import (
	"context"
	"fmt"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// Enqueue is idempotent (EnqueueConversationStartedOutboxEntry and
// EnqueueReplyPostedOutboxEntry are both ON CONFLICT (sequence) DO
// NOTHING inserts): Append already enqueues the outbox row itself, in the
// same transaction as the event, so the use case's subsequent Enqueue
// call is always a safe no-op in production - this method only does real
// work when a caller (a contract test, say) drives Outbox in isolation,
// without a prior Append.
func (s *Store) Enqueue(ctx context.Context, seq domain.Sequence, event domain.Event) error {
	switch e := event.(type) {
	case domain.ConversationStarted:
		if err := s.queries.EnqueueConversationStartedOutboxEntry(ctx, toEnqueueConversationStartedOutboxEntryParams(seq, e)); err != nil {
			return fmt.Errorf("could not enqueue outbox entry: %w", err)
		}
	case domain.ReplyPosted:
		if err := s.queries.EnqueueReplyPostedOutboxEntry(ctx, toEnqueueReplyPostedOutboxEntryParams(seq, e)); err != nil {
			return fmt.Errorf("could not enqueue outbox entry: %w", err)
		}
	default:
		return fmt.Errorf("cannot enqueue event of unrecognized type %T", event)
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
		event, err := toDomainEvent(row)
		if err != nil {
			return nil, err
		}
		entries[i] = out.OutboxEntry{
			Sequence: domain.Sequence(row.Sequence),
			Event:    event,
		}
	}

	return entries, nil
}

func toDomainEvent(row ListPendingOutboxEntriesRow) (domain.Event, error) {
	switch row.EventType {
	case eventTypeConversationStarted:
		return domain.ConversationStarted{
			ConversationID: domain.ConversationID(row.ConversationID),
			ThreadID:       domain.ThreadID(row.ThreadID),
			MessageID:      domain.MessageID(row.MessageID),
			Creator:        domain.CreatorID(row.Creator.String),
			ResourceURL:    domain.ResourceURL(row.ResourceUrl.String),
			ThreadTitle:    domain.ThreadTitle(row.ThreadTitle.String),
			Author:         domain.ParticipantID(row.Author),
			Recipients:     stringsToRecipients(row.Recipients),
			MessageText:    domain.MessageText(row.MessageText),
			OccurredAt:     row.OccurredAt.Time,
		}, nil
	case eventTypeReplyPosted:
		return domain.ReplyPosted{
			ConversationID: domain.ConversationID(row.ConversationID),
			ThreadID:       domain.ThreadID(row.ThreadID),
			MessageID:      domain.MessageID(row.MessageID),
			Author:         domain.ParticipantID(row.Author),
			MessageText:    domain.MessageText(row.MessageText),
			OccurredAt:     row.OccurredAt.Time,
		}, nil
	default:
		return nil, fmt.Errorf("unrecognized event_type %q for outbox sequence %d", row.EventType, row.Sequence)
	}
}

func (s *Store) MarkDone(ctx context.Context, seq domain.Sequence) error {
	if err := s.queries.MarkOutboxEntryDone(ctx, int64(seq)); err != nil {
		return fmt.Errorf("could not mark outbox entry %d done: %w", seq, err)
	}

	return nil
}
