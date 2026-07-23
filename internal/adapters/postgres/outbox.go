package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// Enqueue is idempotent (EnqueueOutboxEntry is an ON CONFLICT (sequence)
// DO NOTHING insert): Append already enqueues the outbox row itself, in
// the same transaction as the event, so a use case's subsequent Enqueue
// call is always a safe no-op in production - this method only does real
// work when a caller (a contract test, say) drives Outbox in isolation,
// without a prior Append.
func (s *Store) Enqueue(ctx context.Context, seq domain.Sequence, event domain.Event) error {
	row, err := marshalEvent(event)
	if err != nil {
		return fmt.Errorf("could not enqueue outbox entry: %w", err)
	}

	if err := s.queries.EnqueueOutboxEntry(ctx, EnqueueOutboxEntryParams{
		Sequence:       int64(seq),
		EventType:      row.EventType,
		ConversationID: string(row.ConversationID),
		OccurredAt:     toTimestamptz(row.OccurredAt),
		Payload:        row.Payload,
	}); err != nil {
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

// toDomainEvent switches on event_type - there's no way around needing to
// know the target type before unmarshalling the payload column.
func toDomainEvent(row ListPendingOutboxEntriesRow) (domain.Event, error) {
	switch row.EventType {
	case eventTypeConversationCreated:
		var payload conversationCreatedPayload
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			return nil, fmt.Errorf("could not unmarshal ConversationCreated payload for outbox sequence %d: %w", row.Sequence, err)
		}
		return domain.ConversationCreated{
			ConversationID: domain.ConversationID(row.ConversationID),
			Creator:        domain.CreatorID(payload.Creator),
			ResourceURL:    domain.ResourceURL(payload.ResourceURL),
			OccurredAt:     row.OccurredAt.Time,
		}, nil
	case eventTypeThreadStarted:
		var payload threadStartedPayload
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			return nil, fmt.Errorf("could not unmarshal ThreadStarted payload for outbox sequence %d: %w", row.Sequence, err)
		}
		return domain.ThreadStarted{
			ConversationID: domain.ConversationID(row.ConversationID),
			ThreadID:       domain.ThreadID(payload.ThreadID),
			ThreadTitle:    domain.ThreadTitle(payload.ThreadTitle),
			Author:         domain.ParticipantID(payload.Author),
			Recipients:     stringsToRecipients(payload.Recipients),
			OccurredAt:     row.OccurredAt.Time,
		}, nil
	case eventTypeMessagePosted:
		var payload messagePostedPayload
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			return nil, fmt.Errorf("could not unmarshal MessagePosted payload for outbox sequence %d: %w", row.Sequence, err)
		}
		return domain.MessagePosted{
			ConversationID: domain.ConversationID(row.ConversationID),
			ThreadID:       domain.ThreadID(payload.ThreadID),
			MessageID:      domain.MessageID(payload.MessageID),
			Author:         domain.ParticipantID(payload.Author),
			MessageText:    domain.MessageText(payload.MessageText),
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
