package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

func Projection(t *testing.T, newProjection func() out.Projection) {
	t.Helper()

	t.Run("checkpoint starts at zero", func(t *testing.T) {
		projection := newProjection()

		checkpoint, err := projection.Checkpoint(context.Background())
		if err != nil {
			t.Fatalf("Checkpoint returned an unexpected error: %v", err)
		}
		if checkpoint != 0 {
			t.Errorf("Checkpoint() before any Apply = %d, want 0", checkpoint)
		}
	})

	t.Run("getting an unknown conversation is ErrConversationNotFound", func(t *testing.T) {
		projection := newProjection()

		_, err := projection.Get(context.Background(), domain.ConversationID("does-not-exist"))
		if !errors.Is(err, domain.ErrConversationNotFound) {
			t.Errorf("Get(unknown) returned err = %v, want domain.ErrConversationNotFound", err)
		}
	})

	t.Run("applying an event makes it readable and advances the checkpoint", func(t *testing.T) {
		projection := newProjection()
		ctx := context.Background()
		event := sampleEvent("conversation-1")

		if err := projection.Apply(ctx, event, 5); err != nil {
			t.Fatalf("Apply returned an unexpected error: %v", err)
		}

		view, err := projection.Get(ctx, event.ConversationID)
		if err != nil {
			t.Fatalf("Get returned an unexpected error: %v", err)
		}
		if view.ResourceURL != event.ResourceURL {
			t.Errorf("Get().ResourceURL = %q, want %q", view.ResourceURL, event.ResourceURL)
		}
		if len(view.Thread.Messages) != 1 || view.Thread.Messages[0].Author != event.Author {
			t.Errorf("Get().Thread.Messages = %#v, want one message from %q", view.Thread.Messages, event.Author)
		}

		checkpoint, err := projection.Checkpoint(ctx)
		if err != nil {
			t.Fatalf("Checkpoint returned an unexpected error: %v", err)
		}
		if checkpoint != 5 {
			t.Errorf("Checkpoint() after Apply(seq=5) = %d, want 5", checkpoint)
		}
	})
}
