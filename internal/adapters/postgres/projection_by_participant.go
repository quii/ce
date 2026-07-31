package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/quii/ce/internal/domain"
)

// GetByParticipant returns all conversations the participant appears in,
// with threads filtered to those the participant is part of, ordered by
// most-recently-active first (rule 3 of "get conversations by participant").
func (s *Store) GetByParticipant(ctx context.Context, id domain.ParticipantID) ([]domain.ConversationView, error) {
	participant := string(id)

	convRows, err := s.queries.ListConversationIDsByParticipant(ctx, participant)
	if err != nil {
		return nil, fmt.Errorf("could not list conversations by participant %q: %w", id, err)
	}

	views := make([]domain.ConversationView, 0, len(convRows))
	for _, row := range convRows {
		convID := row.ConversationID

		convRow, err := s.queries.GetConversationProjection(ctx, convID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("could not get conversation projection %q: %w", convID, err)
		}

		threadRows, err := s.queries.ListParticipantThreadsForConversation(ctx, ListParticipantThreadsForConversationParams{
			ConversationID: convID,
			Column2:        participant,
		})
		if err != nil {
			return nil, fmt.Errorf("could not list participant threads for conversation %q: %w", convID, err)
		}

		msgRows, err := s.queries.ListParticipantMessagesForConversation(ctx, ListParticipantMessagesForConversationParams{
			ConversationID: convID,
			Column2:        participant,
		})
		if err != nil {
			return nil, fmt.Errorf("could not list participant messages for conversation %q: %w", convID, err)
		}

		messagesByThread := make(map[string][]domain.MessageView, len(threadRows))
		for _, m := range msgRows {
			messagesByThread[m.ThreadID] = append(messagesByThread[m.ThreadID], domain.MessageView{
				Author:   domain.ParticipantID(m.Author),
				Text:     domain.MessageText(m.MessageText),
				PostedAt: m.PostedAt.Time,
			})
		}

		threads := make([]domain.ThreadView, len(threadRows))
		for i, t := range threadRows {
			threads[i] = domain.ThreadView{
				ID:           domain.ThreadID(t.ID),
				Title:        domain.ThreadTitle(t.Title),
				Participants: stringsToRecipients(t.Participants),
				Messages:     messagesByThread[t.ID],
			}
		}

		views = append(views, domain.ConversationView{
			ID:          domain.ConversationID(convRow.ID),
			ResourceURL: domain.ResourceURL(convRow.ResourceUrl),
			Threads:     threads,
		})
	}

	return views, nil
}
