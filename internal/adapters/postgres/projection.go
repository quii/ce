package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/quii/ce/internal/domain"
)

// Apply advances both the read model and the checkpoint in one
// transaction, so a reader comparing Checkpoint against a requested
// sequence never observes one move without the other - see
// out.Projection.
func (s *Store) Apply(ctx context.Context, event domain.ConversationStarted, seq domain.Sequence) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not begin projection transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	if err := q.ApplyConversationProjection(ctx, ApplyConversationProjectionParams{
		ID:              string(event.ConversationID),
		ResourceUrl:     string(event.ResourceURL),
		ThreadID:        string(event.ThreadID),
		ThreadTitle:     string(event.ThreadTitle),
		Recipients:      recipientsToStrings(event.Recipients),
		MessageAuthor:   string(event.Author),
		MessageText:     string(event.MessageText),
		MessagePostedAt: toTimestamptz(event.OccurredAt),
	}); err != nil {
		return fmt.Errorf("could not apply conversation projection: %w", err)
	}

	if err := q.SetProjectionCheckpoint(ctx, int64(seq)); err != nil {
		return fmt.Errorf("could not advance projection checkpoint: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit projection transaction: %w", err)
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

	return domain.ConversationView{
		ID:          domain.ConversationID(row.ID),
		ResourceURL: domain.ResourceURL(row.ResourceUrl),
		Thread: domain.ThreadView{
			ID:         domain.ThreadID(row.ThreadID),
			Title:      domain.ThreadTitle(row.ThreadTitle),
			Recipients: stringsToRecipients(row.Recipients),
			Messages: []domain.MessageView{
				{
					Author:   domain.ParticipantID(row.MessageAuthor),
					Text:     domain.MessageText(row.MessageText),
					PostedAt: row.MessagePostedAt.Time,
				},
			},
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
