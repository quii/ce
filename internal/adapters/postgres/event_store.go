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
func (s *Store) Append(ctx context.Context, event domain.Event) (domain.Sequence, error) {
	switch e := event.(type) {
	case domain.ConversationStarted:
		return s.appendConversationStarted(ctx, e)
	case domain.ReplyPosted:
		return s.appendReplyPosted(ctx, e)
	default:
		return 0, fmt.Errorf("cannot append event of unrecognized type %T", event)
	}
}

func (s *Store) appendConversationStarted(ctx context.Context, event domain.ConversationStarted) (domain.Sequence, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not begin append transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	seq, err := q.InsertConversationStartedEvent(ctx, InsertConversationStartedEventParams{
		ConversationID: string(event.ConversationID),
		ThreadID:       string(event.ThreadID),
		MessageID:      string(event.MessageID),
		Creator:        toNullableText(string(event.Creator)),
		ResourceUrl:    toNullableText(string(event.ResourceURL)),
		ThreadTitle:    toNullableText(string(event.ThreadTitle)),
		Author:         string(event.Author),
		Recipients:     recipientsToStrings(event.Recipients),
		MessageText:    string(event.MessageText),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	})
	if err != nil {
		return 0, fmt.Errorf("could not append conversation started event: %w", err)
	}

	if err := q.EnqueueConversationStartedOutboxEntry(ctx, toEnqueueConversationStartedOutboxEntryParams(domain.Sequence(seq), event)); err != nil {
		return 0, fmt.Errorf("could not enqueue outbox entry for appended event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("could not commit append transaction: %w", err)
	}

	return domain.Sequence(seq), nil
}

func (s *Store) appendReplyPosted(ctx context.Context, event domain.ReplyPosted) (domain.Sequence, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not begin append transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	seq, err := q.InsertReplyPostedEvent(ctx, InsertReplyPostedEventParams{
		ConversationID: string(event.ConversationID),
		ThreadID:       string(event.ThreadID),
		MessageID:      string(event.MessageID),
		Author:         string(event.Author),
		MessageText:    string(event.MessageText),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	})
	if err != nil {
		return 0, fmt.Errorf("could not append reply posted event: %w", err)
	}

	if err := q.EnqueueReplyPostedOutboxEntry(ctx, toEnqueueReplyPostedOutboxEntryParams(domain.Sequence(seq), event)); err != nil {
		return 0, fmt.Errorf("could not enqueue outbox entry for appended event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("could not commit append transaction: %w", err)
	}

	return domain.Sequence(seq), nil
}
