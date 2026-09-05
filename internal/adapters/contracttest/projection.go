package contracttest

import (
	"context"
	"testing"
	"time"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

func Projection(t *testing.T, newProjection func() out.Projection) {
	t.Helper()

	t.Run("checkpoint starts at zero", func(t *testing.T) {
		projection := newProjection()

		checkpoint, err := projection.Checkpoint(context.Background())
		assert.NoErr(t, err, "Checkpoint")
		assert.Equal(t, checkpoint, domain.Sequence(0), "Checkpoint() before any Apply")
	})

	t.Run("getting an unknown conversation is ErrConversationNotFound", func(t *testing.T) {
		projection := newProjection()

		_, err := projection.Get(context.Background(), domain.ConversationID("does-not-exist"))
		assert.ErrorIs(t, err, domain.ErrConversationNotFound, "Get(unknown)")
	})

	projectionExistsTests(t, newProjection)

	// A ConversationCreated event on its own describes a conversation with
	// no thread yet - not a state any of this codebase's rules give a
	// representation for (every completed story's Conversation always has
	// exactly one thread). Applying ConversationCreated alone, ahead of the
	// ThreadStarted event that always accompanies it in the same atomic
	// write (docs/adr/0029-fine-grained-events.md), should behave the same
	// as not having applied anything at all yet, not surface a
	// conversation with a zero-value thread.
	t.Run("a conversation isn't readable until its thread has started", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 1, Event: created}), "Apply(ConversationCreated)")

		_, err := projection.Get(ctx, created.ConversationID)
		assert.ErrorIs(t, err, domain.ErrConversationNotFound, "Get() after ConversationCreated alone, before ThreadStarted")
	})

	// TestStartConversation_RaisesThreeEventsAtomically
	// (internal/ports/in/start_conversation_test.go) proves the write side
	// of this; this subtest proves the projection side - applying a batch
	// of events makes the conversation readable and advances the
	// checkpoint once, to the last event's sequence, not once per event -
	// see docs/adr/0029-fine-grained-events.md.
	t.Run("applying a batch of events makes the conversation readable and advances the checkpoint once", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		threadStarted := sampleThreadStarted("conversation-1", "thread-1")
		messagePosted := sampleMessagePosted("conversation-1", "thread-1", "message-1")

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: created},
			out.OutboxEntry{Sequence: 2, Event: threadStarted},
			out.OutboxEntry{Sequence: 3, Event: messagePosted},
		), "Apply(batch)")

		view, err := projection.Get(ctx, created.ConversationID)
		assert.NoErr(t, err, "Get")
		assert.Equal(t, view.ResourceURL, created.ResourceURL, "Get().ResourceURL")
		assert.Len(t, view.Threads, 1, "Get().Threads")
		assert.Equal(t, view.Threads[0].Title, threadStarted.ThreadTitle, "Get().Threads[0].Title")
		assert.Len(t, view.Threads[0].Messages, 1, "Get().Threads[0].Messages")
		assert.Equal(t, view.Threads[0].Messages[0].Author, messagePosted.Author, "Get().Threads[0].Messages[0].Author")

		checkpoint, err := projection.Checkpoint(ctx)
		assert.NoErr(t, err, "Checkpoint")
		assert.Equal(t, checkpoint, domain.Sequence(3), "Checkpoint() after Apply(batch of 3) - advances once, to the last event's sequence")
	})

	t.Run("applying zero entries is a no-op that leaves the checkpoint untouched", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 1, Event: sampleConversationCreated("conversation-1")}), "Apply(one entry)")
		assert.NoErr(t, projection.Apply(ctx), "Apply(no entries)")

		checkpoint, err := projection.Checkpoint(ctx)
		assert.NoErr(t, err, "Checkpoint")
		assert.Equal(t, checkpoint, domain.Sequence(1), "Checkpoint() after Apply(no entries) - must not move")
	})

	t.Run("applying a message appends without disturbing the thread's title or participants", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()

		created := sampleConversationCreated("conversation-1")
		threadStarted := sampleThreadStarted("conversation-1", "thread-1")
		opening := sampleMessagePosted("conversation-1", "thread-1", "message-1")

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: created},
			out.OutboxEntry{Sequence: 2, Event: threadStarted},
			out.OutboxEntry{Sequence: 3, Event: opening},
		), "Apply(batch)")

		reply := domain.MessagePosted{
			ConversationID: created.ConversationID,
			ThreadID:       threadStarted.ThreadID,
			MessageID:      domain.MessageID("message-2"),
			Author:         domain.ParticipantID("user-2"),
			MessageText:    domain.MessageText("Looking into it"),
			OccurredAt:     time.Date(2024, 1, 2, 3, 5, 0, 0, time.UTC),
		}
		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 4, Event: reply}), "Apply(reply)")

		view, err := projection.Get(ctx, created.ConversationID)
		assert.NoErr(t, err, "Get")
		assert.Equal(t, view.Threads[0].Title, threadStarted.ThreadTitle, "Get().Threads[0].Title (unchanged by the reply)")
		assert.Equal(t, view.Threads[0].Participants, threadStarted.Participants(), "Get().Threads[0].Participants (unchanged by the reply)")
		assert.Len(t, view.Threads[0].Messages, 2, "Get().Threads[0].Messages (the opening message plus the reply)")
		assert.Equal(t, view.Threads[0].Messages[0].Author, opening.Author, "Get().Threads[0].Messages[0].Author (the opening author)")
		assert.Equal(t, view.Threads[0].Messages[1].Author, reply.Author, "Get().Threads[0].Messages[1].Author (the replying author)")

		checkpoint, err := projection.Checkpoint(ctx)
		assert.NoErr(t, err, "Checkpoint")
		assert.Equal(t, checkpoint, domain.Sequence(4), "Checkpoint() after applying the batch plus one more event")
	})

	t.Run("applying participant membership events changes only the targeted thread", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()
		created := sampleConversationCreated("conversation-participants")
		thread := sampleThreadStarted("conversation-participants", "thread-participants")
		opening := sampleMessagePosted("conversation-participants", "thread-participants", "message-participants")

		assert.NoErr(t, projection.Apply(ctx,
			out.OutboxEntry{Sequence: 1, Event: created},
			out.OutboxEntry{Sequence: 2, Event: thread},
			out.OutboxEntry{Sequence: 3, Event: opening},
			out.OutboxEntry{Sequence: 4, Event: domain.ParticipantAdded{ConversationID: created.ConversationID, ThreadID: thread.ThreadID, ParticipantID: "user-4"}},
		), "Apply(ParticipantAdded)")

		view, err := projection.Get(ctx, created.ConversationID)
		assert.NoErr(t, err, "Get after ParticipantAdded")
		assert.Equal(t, view.Threads[0].Participants, append(thread.Participants(), "user-4"), "Get().Threads[0].Participants after ParticipantAdded")

		assert.NoErr(t, projection.Apply(ctx, out.OutboxEntry{Sequence: 5, Event: domain.ParticipantRemoved{ConversationID: created.ConversationID, ThreadID: thread.ThreadID, ParticipantID: "user-1"}}), "Apply(ParticipantRemoved)")
		view, err = projection.Get(ctx, created.ConversationID)
		assert.NoErr(t, err, "Get after ParticipantRemoved")
		assert.Equal(t, view.Threads[0].Participants, domain.Recipients{"user-2", "user-3", "user-4"}, "Get().Threads[0].Participants after ParticipantRemoved")
	})

	projectionThreadTests(t, newProjection)
}
