package specifications

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

// ConversationDriver is the in-port surface a driver has to implement to
// run ConversationSpecification - see docs/adr/0022-specifications-and-drivers.md.
type ConversationDriver interface {
	in.ConversationStarter
	in.ConversationGetter
}

// ConversationSpecification covers rules 1-4 of the "start a conversation"
// story - field presence, the recipients set, and author/recipient
// exclusivity - plus the two reads that never depend on the projection
// having caught up (rules 8-9). "Once caught up" behaviour (rules 5-7, 10)
// needs explicit control over the relay's Drain, which an HTTP-only
// driver has no way to expose - see ConversationProjectionSpecification.
func ConversationSpecification(t *testing.T, driver ConversationDriver) {
	t.Helper()

	validCommand := func() in.StartConversationCommand {
		return in.StartConversationCommand{
			ResourceURL: strPtr("https://example.com/orders/123"),
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{"user-2", "user-3"},
			Message:     strPtr("Where is my order?"),
		}
	}

	t.Run("a missing resource url is rejected", func(t *testing.T) {
		cmd := validCommand()
		cmd.ResourceURL = nil
		assertStartRejected(t, driver, cmd)
	})

	t.Run("a missing thread title is rejected", func(t *testing.T) {
		cmd := validCommand()
		cmd.ThreadTitle = nil
		assertStartRejected(t, driver, cmd)
	})

	t.Run("a missing author is rejected", func(t *testing.T) {
		cmd := validCommand()
		cmd.Author = nil
		assertStartRejected(t, driver, cmd)
	})

	t.Run("a missing opening message is rejected", func(t *testing.T) {
		cmd := validCommand()
		cmd.Message = nil
		assertStartRejected(t, driver, cmd)
	})

	t.Run("a missing recipients field is rejected", func(t *testing.T) {
		cmd := validCommand()
		cmd.Recipients = nil
		assertStartRejected(t, driver, cmd)
	})

	t.Run("duplicate recipient ids are rejected", func(t *testing.T) {
		cmd := validCommand()
		cmd.Recipients = &[]string{"user-2", "user-2"}
		assertStartRejected(t, driver, cmd)
	})

	t.Run("an author who is also listed as a recipient is rejected", func(t *testing.T) {
		cmd := validCommand()
		cmd.Author = strPtr("user-1")
		cmd.Recipients = &[]string{"user-1", "user-2"}
		assertStartRejected(t, driver, cmd)
	})

	t.Run("reading a conversation before the projection has caught up is pending", func(t *testing.T) {
		ctx := context.Background()

		started, err := driver.StartConversation(ctx, validCommand())
		assert.NoErr(t, err, "StartConversation")

		after := int64(started.Sequence)
		_, err = driver.GetConversation(ctx, in.GetConversationCommand{
			ConversationID: string(started.ConversationID),
			After:          &after,
		})
		assert.ErrorIs(t, err, domain.ErrProjectionNotCaughtUp, "GetConversation(after=%d) immediately after the write", after)
	})

	t.Run("a plain read reflects whatever the projection currently holds", func(t *testing.T) {
		ctx := context.Background()

		started, err := driver.StartConversation(ctx, validCommand())
		assert.NoErr(t, err, "StartConversation")

		_, err = driver.GetConversation(ctx, in.GetConversationCommand{ConversationID: string(started.ConversationID)})
		assert.ErrorIs(t, err, domain.ErrConversationNotFound, "plain GetConversation before any projection has run")
	})
}

func assertStartRejected(t *testing.T, driver ConversationDriver, cmd in.StartConversationCommand) {
	t.Helper()

	_, err := driver.StartConversation(context.Background(), cmd)
	_ = assert.ErrorAs[domain.ValidationError](t, err, "StartConversation(%+v)", cmd)
}

func strPtr(s string) *string { return &s }
