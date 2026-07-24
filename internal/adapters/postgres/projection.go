package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"

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
			if err := applyThreadStarted(ctx, q, e, entry.Sequence); err != nil {
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

// applyThreadStarted inserts a new row into conversation_projection_threads,
// keyed by the thread's own id and ordered by seq - the global event
// sequence this ThreadStarted was applied at - so ListConversationProjectionThreads
// can recover creation order (rule 10 of "add a thread to a conversation")
// without a conversation being limited to a single thread any more (rule 9).
func applyThreadStarted(ctx context.Context, q *Queries, event domain.ThreadStarted, seq domain.Sequence) error {
	if err := q.ApplyThreadStartedProjection(ctx, ApplyThreadStartedProjectionParams{
		ID:             string(event.ThreadID),
		ConversationID: string(event.ConversationID),
		Sequence:       int64(seq),
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
		ThreadID:       string(event.ThreadID),
		Sequence:       int64(seq),
		Author:         string(event.Author),
		MessageText:    string(event.MessageText),
		PostedAt:       toTimestamptz(event.OccurredAt),
	}); err != nil {
		return fmt.Errorf("could not append message to projection: %w", err)
	}

	return nil
}

// Get treats a conversation with no threads yet - either wholly unknown,
// or a ConversationCreated applied without any ThreadStarted companion -
// as not found: an empty ListConversationProjectionThreads result is the
// one signal used for that, the same test the memory adapter's Get uses,
// rather than a separate existence subquery duplicating what the thread
// list already yields for free. The three reads a full Get needs - the
// conversation's own row, its threads, and its messages - are independent
// of each other, so they run concurrently rather than as three sequential
// round trips. This concurrency is a deliberate, acknowledged exception to
// docs/adr/0013-implement-only-the-current-test.md: no scenario forces it
// (a sequential version passes every existing test identically), it's kept
// anyway for the latency win on the busiest read path in the service.
func (s *Store) Get(ctx context.Context, id domain.ConversationID) (domain.ConversationView, error) {
	var (
		conversationRow                          ConversationProjection
		threadRows                               []ListConversationProjectionThreadsRow
		messageRows                              []ListConversationProjectionMessagesRow
		conversationErr, threadsErr, messagesErr error
	)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		conversationRow, conversationErr = s.queries.GetConversationProjection(ctx, string(id))
	}()
	go func() {
		defer wg.Done()
		threadRows, threadsErr = s.queries.ListConversationProjectionThreads(ctx, string(id))
	}()
	go func() {
		defer wg.Done()
		messageRows, messagesErr = s.queries.ListConversationProjectionMessages(ctx, string(id))
	}()
	wg.Wait()

	if threadsErr != nil {
		return domain.ConversationView{}, fmt.Errorf("could not list conversation projection threads %q: %w", id, threadsErr)
	}
	if len(threadRows) == 0 {
		return domain.ConversationView{}, domain.ErrConversationNotFound
	}

	if conversationErr != nil {
		if errors.Is(conversationErr, pgx.ErrNoRows) {
			return domain.ConversationView{}, domain.ErrConversationNotFound
		}
		return domain.ConversationView{}, fmt.Errorf("could not get conversation projection %q: %w", id, conversationErr)
	}
	if messagesErr != nil {
		return domain.ConversationView{}, fmt.Errorf("could not list conversation projection messages %q: %w", id, messagesErr)
	}

	messagesByThread := make(map[string][]domain.MessageView, len(threadRows))
	for _, m := range messageRows {
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

	return domain.ConversationView{
		ID:          domain.ConversationID(conversationRow.ID),
		ResourceURL: domain.ResourceURL(conversationRow.ResourceUrl),
		Threads:     threads,
	}, nil
}

// Exists reports whether a conversation is currently readable - the same
// "has at least one thread" test Get's not-found check uses, without
// paying for a full Get (threads and messages both) when a caller only
// needs a yes/no answer.
func (s *Store) Exists(ctx context.Context, id domain.ConversationID) (bool, error) {
	exists, err := s.queries.ConversationProjectionExists(ctx, string(id))
	if err != nil {
		return false, fmt.Errorf("could not check conversation projection existence %q: %w", id, err)
	}

	return exists, nil
}

func (s *Store) Checkpoint(ctx context.Context) (domain.Sequence, error) {
	seq, err := s.queries.GetProjectionCheckpoint(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not get projection checkpoint: %w", err)
	}

	return domain.Sequence(seq), nil
}
