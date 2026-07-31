package contracttest

import (
	"context"
	"testing"
	"time"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// ProjectionByParticipant covers out.Projection.GetByParticipant - the
// "get conversations by participant" story's projection rules - split into
// its own file to keep projection.go under the file-length limit
// (docs/adr/0004-file-length.md).
func ProjectionByParticipant(t *testing.T, newProjection func() out.Projection) {
	t.Helper()

	t.Run("returns empty slice for unknown participant", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		views, err := projection.GetByParticipant(ctx, domain.ParticipantID("nobody"))
		assert.NoErr(t, err, "GetByParticipant(unknown)")
		assert.Len(t, views, 0, "GetByParticipant(unknown)")
	})

	t.Run("returns only threads the participant is on, not other threads", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conv-filter-1")
		thread1 := domain.ThreadStarted{
			ConversationID: created.ConversationID,
			ThreadID:       "thread-filter-1",
			ThreadTitle:    "Thread with participant",
			Author:         "participant-a",
			Recipients:     domain.Recipients{},
			OccurredAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		msg1 := domain.MessagePosted{
			ConversationID: created.ConversationID,
			ThreadID:       thread1.ThreadID,
			MessageID:      "msg-filter-1",
			Author:         "participant-a",
			MessageText:    "Hello",
			OccurredAt:     time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
		}
		thread2 := domain.ThreadStarted{
			ConversationID: created.ConversationID,
			ThreadID:       "thread-filter-2",
			ThreadTitle:    "Thread without participant",
			Author:         "other-user",
			Recipients:     domain.Recipients{},
			OccurredAt:     time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		}
		msg2 := domain.MessagePosted{
			ConversationID: created.ConversationID,
			ThreadID:       thread2.ThreadID,
			MessageID:      "msg-filter-2",
			Author:         "other-user",
			MessageText:    "Other thread",
			OccurredAt:     time.Date(2024, 1, 1, 0, 1, 1, 0, time.UTC),
		}

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: created},
			out.OutboxEntry{Sequence: 2, Event: thread1},
			out.OutboxEntry{Sequence: 3, Event: msg1},
			out.OutboxEntry{Sequence: 4, Event: thread2},
			out.OutboxEntry{Sequence: 5, Event: msg2},
		), "Apply(batch)")

		views, err := projection.GetByParticipant(ctx, domain.ParticipantID("participant-a"))
		assert.NoErr(t, err, "GetByParticipant")
		assert.Len(t, views, 1, "GetByParticipant - one conversation")
		assert.Len(t, views[0].Threads, 1, "GetByParticipant - one thread (filtered)")
		assert.Equal(t, views[0].Threads[0].Title, thread1.ThreadTitle, "GetByParticipant - thread title")
	})

	t.Run("omits conversations where participant is on no threads", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conv-omit-1")
		thread := sampleThreadStarted("conv-omit-1", "thread-omit-1")
		msg := sampleMessagePosted("conv-omit-1", "thread-omit-1", "msg-omit-1")

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: created},
			out.OutboxEntry{Sequence: 2, Event: thread},
			out.OutboxEntry{Sequence: 3, Event: msg},
		), "Apply(batch)")

		views, err := projection.GetByParticipant(ctx, domain.ParticipantID("non-participant"))
		assert.NoErr(t, err, "GetByParticipant")
		assert.Len(t, views, 0, "GetByParticipant - conversation omitted for non-participant")
	})

	t.Run("orders by most recently active thread, most recent first", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		createdA := sampleConversationCreated("conv-order-a")
		threadA := domain.ThreadStarted{
			ConversationID: createdA.ConversationID,
			ThreadID:       "thread-order-a",
			ThreadTitle:    "Older conv",
			Author:         "order-participant",
			Recipients:     domain.Recipients{},
			OccurredAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		msgA := domain.MessagePosted{
			ConversationID: createdA.ConversationID,
			ThreadID:       threadA.ThreadID,
			MessageID:      "msg-order-a",
			Author:         "order-participant",
			MessageText:    "Earlier",
			OccurredAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		}

		createdB := sampleConversationCreated("conv-order-b")
		threadB := domain.ThreadStarted{
			ConversationID: createdB.ConversationID,
			ThreadID:       "thread-order-b",
			ThreadTitle:    "Newer conv",
			Author:         "order-participant",
			Recipients:     domain.Recipients{},
			OccurredAt:     time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
		}
		msgB := domain.MessagePosted{
			ConversationID: createdB.ConversationID,
			ThreadID:       threadB.ThreadID,
			MessageID:      "msg-order-b",
			Author:         "order-participant",
			MessageText:    "Later",
			OccurredAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		}

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: createdA},
			out.OutboxEntry{Sequence: 2, Event: threadA},
			out.OutboxEntry{Sequence: 3, Event: msgA},
			out.OutboxEntry{Sequence: 4, Event: createdB},
			out.OutboxEntry{Sequence: 5, Event: threadB},
			out.OutboxEntry{Sequence: 6, Event: msgB},
		), "Apply(batch)")

		views, err := projection.GetByParticipant(ctx, domain.ParticipantID("order-participant"))
		assert.NoErr(t, err, "GetByParticipant")
		assert.Len(t, views, 2, "GetByParticipant - two conversations")
		assert.Equal(t, views[0].ID, createdB.ConversationID, "GetByParticipant - newer conversation is first")
		assert.Equal(t, views[1].ID, createdA.ConversationID, "GetByParticipant - older conversation is second")
	})
}
