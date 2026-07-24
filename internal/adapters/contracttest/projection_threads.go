package contracttest

import (
	"context"
	"testing"
	"time"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// projectionExistsTests covers out.Projection.Exists, added by "add a
// thread to a conversation" for AddThread's cheap existence check (rule 4)
// - split into its own file, alongside projectionThreadTests below, to
// keep projection.go under the file-length limit
// (docs/adr/0004-file-length.md).
func projectionExistsTests(t *testing.T, newProjection func() out.Projection) {
	t.Helper()

	t.Run("Exists reports false for an unknown conversation", func(t *testing.T) {
		projection := newProjection()

		exists, err := projection.Exists(context.Background(), domain.ConversationID("does-not-exist"))
		assert.NoErr(t, err, "Exists(unknown)")
		assert.False(t, exists, "Exists(unknown)")
	})

	t.Run("Exists reports false for a conversation whose thread hasn't started yet", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 1, Event: created}), "Apply(ConversationCreated)")

		exists, err := projection.Exists(ctx, created.ConversationID)
		assert.NoErr(t, err, "Exists() after ConversationCreated alone, before ThreadStarted")
		assert.False(t, exists, "Exists() after ConversationCreated alone, before ThreadStarted")
	})

	t.Run("Exists reports true once a conversation's thread has started", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		threadStarted := sampleThreadStarted("conversation-1", "thread-1")

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: created},
			out.OutboxEntry{Sequence: 2, Event: threadStarted},
		), "Apply(batch)")

		exists, err := projection.Exists(ctx, created.ConversationID)
		assert.NoErr(t, err, "Exists() after ThreadStarted")
		assert.True(t, exists, "Exists() after ThreadStarted")
	})
}

// projectionThreadTests covers rules 9-10 of "add a thread to a
// conversation" at the out-port level, against both the in-memory fake
// and the real Postgres adapter: a conversation's representation is a
// list of threads, in the order each ThreadStarted event was applied
// (not the order Apply happened to be called in), and a message lands
// against the thread its MessagePosted event names, not just whichever
// thread happens to be first.
func projectionThreadTests(t *testing.T, newProjection func() out.Projection) {
	t.Helper()

	t.Run("a conversation can have more than one thread, in creation order, each with its own messages", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		firstThread := sampleThreadStarted("conversation-1", "thread-1")
		firstMessage := sampleMessagePosted("conversation-1", "thread-1", "message-1")

		secondThread := domain.ThreadStarted{
			ConversationID: created.ConversationID,
			ThreadID:       "thread-2",
			ThreadTitle:    "Delivery date",
			Author:         "user-3",
			Recipients:     domain.Recipients{"user-4"},
			OccurredAt:     time.Date(2024, 1, 2, 3, 6, 0, 0, time.UTC),
		}
		secondMessage := domain.MessagePosted{
			ConversationID: created.ConversationID,
			ThreadID:       secondThread.ThreadID,
			MessageID:      "message-2",
			Author:         "user-3",
			MessageText:    "When will this ship?",
			OccurredAt:     time.Date(2024, 1, 2, 3, 6, 1, 0, time.UTC),
		}

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: created},
			out.OutboxEntry{Sequence: 2, Event: firstThread},
			out.OutboxEntry{Sequence: 3, Event: firstMessage},
			out.OutboxEntry{Sequence: 4, Event: secondThread},
			out.OutboxEntry{Sequence: 5, Event: secondMessage},
		), "Apply(batch with two threads)")

		view, err := projection.Get(ctx, created.ConversationID)
		assert.NoErr(t, err, "Get")
		assert.Len(t, view.Threads, 2, "Get().Threads")

		assert.Equal(t, view.Threads[0].Title, firstThread.ThreadTitle, "Get().Threads[0].Title")
		assert.Len(t, view.Threads[0].Messages, 1, "Get().Threads[0].Messages")
		assert.Equal(t, view.Threads[0].Messages[0].Author, firstMessage.Author, "Get().Threads[0].Messages[0].Author")

		assert.Equal(t, view.Threads[1].Title, secondThread.ThreadTitle, "Get().Threads[1].Title")
		assert.Len(t, view.Threads[1].Messages, 1, "Get().Threads[1].Messages")
		assert.Equal(t, view.Threads[1].Messages[0].Author, secondMessage.Author, "Get().Threads[1].Messages[0].Author")
	})

	// Threads are ordered by the sequence they were started at (rule 10 of
	// "add a thread to a conversation"), not by the order Apply happened to
	// be invoked in - applied here deliberately out of sequence order (the
	// higher-sequence thread first) to prove neither adapter is silently
	// relying on apply-call order to produce the right result.
	t.Run("threads are ordered by sequence, not by the order Apply was called in", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 1, Event: created}), "Apply(ConversationCreated)")

		laterThread := domain.ThreadStarted{
			ConversationID: created.ConversationID,
			ThreadID:       "thread-later",
			ThreadTitle:    "Second topic",
			Author:         "user-1",
			Recipients:     domain.Recipients{},
			OccurredAt:     time.Date(2024, 1, 2, 3, 6, 0, 0, time.UTC),
		}
		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 3, Event: laterThread}), "Apply(higher-sequence thread, applied first)")

		earlierThread := domain.ThreadStarted{
			ConversationID: created.ConversationID,
			ThreadID:       "thread-earlier",
			ThreadTitle:    "First topic",
			Author:         "user-1",
			Recipients:     domain.Recipients{},
			OccurredAt:     time.Date(2024, 1, 2, 3, 5, 0, 0, time.UTC),
		}
		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 2, Event: earlierThread}), "Apply(lower-sequence thread, applied second)")

		view, err := projection.Get(ctx, created.ConversationID)
		assert.NoErr(t, err, "Get")
		assert.Len(t, view.Threads, 2, "Get().Threads")
		assert.Equal(t, view.Threads[0].Title, earlierThread.ThreadTitle, "Get().Threads[0].Title - ordered by sequence (2), despite being applied second")
		assert.Equal(t, view.Threads[1].Title, laterThread.ThreadTitle, "Get().Threads[1].Title - ordered by sequence (3), despite being applied first")
	})
}
