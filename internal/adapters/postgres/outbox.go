package postgres

import (
	"context"
	"fmt"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// Enqueue is idempotent (every EnqueueXOutboxEntry query is an ON CONFLICT
// (sequence) DO NOTHING insert): Append already enqueues the outbox row
// itself, in the same transaction as the event, so a use case's
// subsequent Enqueue call is always a safe no-op in production - this
// method only does real work when a caller (a contract test, say) drives
// Outbox in isolation, without a prior Append.
func (s *Store) Enqueue(ctx context.Context, seq domain.Sequence, event domain.Event) error {
	switch e := event.(type) {
	case domain.ConversationCreated:
		if err := s.queries.EnqueueConversationCreatedOutboxEntry(ctx, toEnqueueConversationCreatedOutboxEntryParams(seq, e)); err != nil {
			return fmt.Errorf("could not enqueue outbox entry: %w", err)
		}
	case domain.ThreadStarted:
		if err := s.queries.EnqueueThreadStartedOutboxEntry(ctx, toEnqueueThreadStartedOutboxEntryParams(seq, e)); err != nil {
			return fmt.Errorf("could not enqueue outbox entry: %w", err)
		}
	case domain.MessagePosted:
		if err := s.queries.EnqueueMessagePostedOutboxEntry(ctx, toEnqueueMessagePostedOutboxEntryParams(seq, e)); err != nil {
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
	case eventTypeConversationCreated:
		return domain.ConversationCreated{
			ConversationID: domain.ConversationID(row.ConversationID),
			Creator:        domain.CreatorID(row.Creator.String),
			ResourceURL:    domain.ResourceURL(row.ResourceUrl.String),
			OccurredAt:     row.OccurredAt.Time,
		}, nil
	case eventTypeThreadStarted:
		return domain.ThreadStarted{
			ConversationID: domain.ConversationID(row.ConversationID),
			ThreadID:       domain.ThreadID(row.ThreadID.String),
			ThreadTitle:    domain.ThreadTitle(row.ThreadTitle.String),
			Author:         domain.ParticipantID(row.Author.String),
			Recipients:     stringsToRecipients(row.Recipients),
			OccurredAt:     row.OccurredAt.Time,
		}, nil
	case eventTypeMessagePosted:
		return domain.MessagePosted{
			ConversationID: domain.ConversationID(row.ConversationID),
			ThreadID:       domain.ThreadID(row.ThreadID.String),
			MessageID:      domain.MessageID(row.MessageID.String),
			Author:         domain.ParticipantID(row.Author.String),
			MessageText:    domain.MessageText(row.MessageText.String),
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
