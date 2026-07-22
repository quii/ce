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
func (s *Store) Apply(ctx context.Context, event domain.Event, seq domain.Sequence) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not begin projection transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	switch e := event.(type) {
	case domain.ConversationStarted:
		if err := applyConversationStarted(ctx, q, e, seq); err != nil {
			return err
		}
	case domain.ReplyPosted:
		if err := applyReplyPosted(ctx, q, e, seq); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot apply event of unrecognized type %T", event)
	}

	if err := q.SetProjectionCheckpoint(ctx, int64(seq)); err != nil {
		return fmt.Errorf("could not advance projection checkpoint: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit projection transaction: %w", err)
	}

	return nil
}

func applyConversationStarted(ctx context.Context, q *Queries, event domain.ConversationStarted, seq domain.Sequence) error {
	if err := q.ApplyConversationStartedProjection(ctx, ApplyConversationStartedProjectionParams{
		ID:           string(event.ConversationID),
		ResourceUrl:  string(event.ResourceURL),
		ThreadID:     string(event.ThreadID),
		ThreadTitle:  string(event.ThreadTitle),
		Participants: recipientsToStrings(event.Participants()),
	}); err != nil {
		return fmt.Errorf("could not apply conversation started projection: %w", err)
	}

	if err := q.AppendConversationProjectionMessage(ctx, AppendConversationProjectionMessageParams{
		ConversationID: string(event.ConversationID),
		Sequence:       int64(seq),
		Author:         string(event.Author),
		MessageText:    string(event.MessageText),
		PostedAt:       toTimestamptz(event.OccurredAt),
	}); err != nil {
		return fmt.Errorf("could not append opening message to projection: %w", err)
	}

	return nil
}

func applyReplyPosted(ctx context.Context, q *Queries, event domain.ReplyPosted, seq domain.Sequence) error {
	if err := q.AppendConversationProjectionMessage(ctx, AppendConversationProjectionMessageParams{
		ConversationID: string(event.ConversationID),
		Sequence:       int64(seq),
		Author:         string(event.Author),
		MessageText:    string(event.MessageText),
		PostedAt:       toTimestamptz(event.OccurredAt),
	}); err != nil {
		return fmt.Errorf("could not append reply to projection: %w", err)
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
