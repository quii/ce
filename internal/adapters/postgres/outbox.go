package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

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
		event, err := toDomainEvent(row.Sequence, row.EventType, row.ConversationID, row.OccurredAt, row.Payload)
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
// know the target type before unmarshalling the payload column. Takes the
// shared columns as plain values rather than a specific sqlc-generated row
// type, since conversation_events and conversation_outbox both store the
// same shape (event_type, conversation_id, occurred_at, payload) under
// different generated Go types - shared here (event_store.go's
// ListByConversation and this file's Pending both call it) rather than
// duplicating the switch/unmarshal per row type.
func toDomainEvent(sequence int64, eventType, conversationID string, occurredAt pgtype.Timestamptz, payload []byte) (domain.Event, error) {
	switch eventType {
	case eventTypeConversationCreated:
		var p conversationCreatedPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("could not unmarshal ConversationCreated payload for sequence %d: %w", sequence, err)
		}
		return domain.ConversationCreated{
			ConversationID: domain.ConversationID(conversationID),
			Creator:        domain.CreatorID(p.Creator),
			ResourceURL:    domain.ResourceURL(p.ResourceURL),
			OccurredAt:     occurredAt.Time,
		}, nil
	case eventTypeThreadStarted:
		var p threadStartedPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("could not unmarshal ThreadStarted payload for sequence %d: %w", sequence, err)
		}
		return domain.ThreadStarted{
			ConversationID: domain.ConversationID(conversationID),
			ThreadID:       domain.ThreadID(p.ThreadID),
			ThreadTitle:    domain.ThreadTitle(p.ThreadTitle),
			Author:         domain.ParticipantID(p.Author),
			Recipients:     stringsToRecipients(p.Recipients),
			OccurredAt:     occurredAt.Time,
		}, nil
	case eventTypeMessagePosted:
		var p messagePostedPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("could not unmarshal MessagePosted payload for sequence %d: %w", sequence, err)
		}
		return domain.MessagePosted{
			ConversationID: domain.ConversationID(conversationID),
			ThreadID:       domain.ThreadID(p.ThreadID),
			MessageID:      domain.MessageID(p.MessageID),
			Author:         domain.ParticipantID(p.Author),
			MessageText:    domain.MessageText(p.MessageText),
			OccurredAt:     occurredAt.Time,
		}, nil
	default:
		return nil, fmt.Errorf("unrecognized event_type %q for sequence %d", eventType, sequence)
	}
}

func (s *Store) MarkDone(ctx context.Context, seq domain.Sequence) error {
	if err := s.queries.MarkOutboxEntryDone(ctx, int64(seq)); err != nil {
		return fmt.Errorf("could not mark outbox entry %d done: %w", seq, err)
	}

	return nil
}
