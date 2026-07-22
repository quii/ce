package contracttest

import (
	"context"
	"testing"

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

	t.Run("applying an event makes it readable and advances the checkpoint", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()
		event := sampleEvent("conversation-1")

		assert.NoErr(t, projection.Apply(ctx, event, 5), "Apply")

		view, err := projection.Get(ctx, event.ConversationID)
		assert.NoErr(t, err, "Get")
		assert.Equal(t, view.ResourceURL, event.ResourceURL, "Get().ResourceURL")
		assert.Len(t, view.Thread.Messages, 1, "Get().Thread.Messages")
		assert.Equal(t, view.Thread.Messages[0].Author, event.Author, "Get().Thread.Messages[0].Author")

		checkpoint, err := projection.Checkpoint(ctx)
		assert.NoErr(t, err, "Checkpoint")
		assert.Equal(t, checkpoint, domain.Sequence(5), "Checkpoint() after Apply(seq=5)")
	})

	t.Run("applying a reply appends a message without disturbing the thread's title or participants", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()
		started := sampleEvent("conversation-1")

		assert.NoErr(t, projection.Apply(ctx, started, 1), "Apply(ConversationStarted)")

		reply := sampleReplyEvent("conversation-1", string(started.ThreadID))
		assert.NoErr(t, projection.Apply(ctx, reply, 2), "Apply(ReplyPosted)")

		view, err := projection.Get(ctx, started.ConversationID)
		assert.NoErr(t, err, "Get")
		assert.Equal(t, view.Thread.Title, started.ThreadTitle, "Get().Thread.Title (unchanged by the reply)")
		assert.Equal(t, view.Thread.Participants, started.Participants(), "Get().Thread.Participants (unchanged by the reply)")
		assert.Len(t, view.Thread.Messages, 2, "Get().Thread.Messages (the opening message plus the reply)")
		assert.Equal(t, view.Thread.Messages[0].Author, started.Author, "Get().Thread.Messages[0].Author (the opening author)")
		assert.Equal(t, view.Thread.Messages[1].Author, reply.Author, "Get().Thread.Messages[1].Author (the replying author)")

		checkpoint, err := projection.Checkpoint(ctx)
		assert.NoErr(t, err, "Checkpoint")
		assert.Equal(t, checkpoint, domain.Sequence(2), "Checkpoint() after applying two events")
	})
}
