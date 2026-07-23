package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// Apply applies every entry in the batch, in one transaction, before
// advancing the checkpoint once - to the last entry's sequence - rather
// than once per entry, so a reader comparing Checkpoint against a
// requested sequence never observes a batch partially applied (e.g. a
// ThreadStarted with no MessagePosted companion yet), per
// docs/adr/0029-fine-grained-events.md.
func (s *Store) Apply(ctx context.Context, entries ...out.OutboxEntry) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not begin projection transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	for _, entry := range entries {
		switch e := entry.Event.(type) {
		case domain.ConversationCreated:
			if err := applyConversationCreated(ctx, q, e); err != nil {
				return err
			}
		case domain.ThreadStarted:
			if err := applyThreadStarted(ctx, q, e); err != nil {
				return err
			}
		case domain.MessagePosted:
			if err := applyMessagePosted(ctx, q, e, entry.Sequence); err != nil {
				return err
			}
		default:
			return fmt.Errorf("cannot apply event of unrecognized type %T", entry.Event)
		}
	}

	if len(entries) > 0 {
		if err := q.SetProjectionCheckpoint(ctx, int64(entries[len(entries)-1].Sequence)); err != nil {
			return fmt.Errorf("could not advance projection checkpoint: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit projection transaction: %w", err)
	}

	return nil
}

func applyConversationCreated(ctx context.Context, q *Queries, event domain.ConversationCreated) error {
	if err := q.ApplyConversationCreatedProjection(ctx, ApplyConversationCreatedProjectionParams{
		ID:          string(event.ConversationID),
		ResourceUrl: string(event.ResourceURL),
	}); err != nil {
		return fmt.Errorf("could not apply conversation created projection: %w", err)
	}

	return nil
}

func applyThreadStarted(ctx context.Context, q *Queries, event domain.ThreadStarted) error {
	if err := q.ApplyThreadStartedProjection(ctx, ApplyThreadStartedProjectionParams{
		ID:             string(event.ThreadID),
		ConversationID: string(event.ConversationID),
		Title:          string(event.ThreadTitle),
		Participants:   recipientsToStrings(event.Participants()),
	}); err != nil {
		return fmt.Errorf("could not apply thread started projection: %w", err)
	}

	return nil
}

func applyMessagePosted(ctx context.Context, q *Queries, event domain.MessagePosted, seq domain.Sequence) error {
	if err := q.AppendConversationProjectionMessage(ctx, AppendConversationProjectionMessageParams{
		ConversationID: string(event.ConversationID),
		Sequence:       int64(seq),
		Author:         string(event.Author),
		MessageText:    string(event.MessageText),
		PostedAt:       toTimestamptz(event.OccurredAt),
	}); err != nil {
		return fmt.Errorf("could not append message to projection: %w", err)
	}

	return nil
}

func (s *Store) Get(ctx context.Context, id domain.ConversationID) (domain.ConversationView, error) {
	row, err := s.queries.GetConversationProjection(ctx, string(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ConversationView{}, domain.ErrConversationNotFound
		}
		return domain.ConversationView{}, fmt.Errorf("could not get conversation projection %q: %w", id, err)
	}

	messageRows, err := s.queries.ListConversationProjectionMessages(ctx, string(id))
	if err != nil {
		return domain.ConversationView{}, fmt.Errorf("could not list conversation projection messages %q: %w", id, err)
	}

	messages := make([]domain.MessageView, len(messageRows))
	for i, m := range messageRows {
		messages[i] = domain.MessageView{
			Author:   domain.ParticipantID(m.Author),
			Text:     domain.MessageText(m.MessageText),
			PostedAt: m.PostedAt.Time,
		}
	}

	return domain.ConversationView{
		ID:          domain.ConversationID(row.ID),
		ResourceURL: domain.ResourceURL(row.ResourceUrl),
		Thread: domain.ThreadView{
			ID:           domain.ThreadID(row.ThreadID),
			Title:        domain.ThreadTitle(row.ThreadTitle),
			Participants: stringsToRecipients(row.Participants),
			Messages:     messages,
		},
	}, nil
}

func (s *Store) Checkpoint(ctx context.Context) (domain.Sequence, error) {
	seq, err := s.queries.GetProjectionCheckpoint(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not get projection checkpoint: %w", err)
	}

	return domain.Sequence(seq), nil
}
