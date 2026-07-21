//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/adapters/postgres/postgrestest"
	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/domain"
)

// TestStore_TimestampsAreUTC guards against the two-time.Time-values-for-
// the-same-instant-fail-equality trap docs/adr/0015-utc-always.md exists
// to prevent: pgx's timestamptz codec builds scanned values via
// time.Unix, which defaults to time.Local, unless ScanLocation is
// explicitly configured on the connection's type map.
func TestStore_TimestampsAreUTC(t *testing.T) {
	connString := postgrestest.StartContainer(t)

	pool, err := postgres.NewPool(context.Background(), connString)
	assert.NoErr(t, err, "connect to postgres")
	t.Cleanup(pool.Close)

	store := postgres.NewStore(pool)
	ctx := context.Background()

	occurredAt := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	event := domain.ConversationStarted{
		ConversationID: "conversation-utc",
		ThreadID:       "thread-utc",
		MessageID:      "message-utc",
		Creator:        domain.PlaceholderCreator,
		ResourceURL:    "https://example.com/orders/utc",
		ThreadTitle:    "UTC check",
		Author:         "user-1",
		Recipients:     domain.Recipients{},
		MessageText:    "does this come back as UTC?",
		OccurredAt:     occurredAt,
	}

	seq, err := store.Append(ctx, event)
	assert.NoErr(t, err, "Append")

	pending, err := store.Pending(ctx)
	assert.NoErr(t, err, "Pending")
	assert.Len(t, pending, 1, "Pending()")
	got, ok := pending[0].Event.(domain.ConversationStarted)
	if !ok {
		t.Fatalf("Pending()[0].Event = %#v, want a domain.ConversationStarted", pending[0].Event)
	}
	// *time.Location has unexported internals cmp.Diff can't traverse -
	// pointer identity (what *Location == compares) is the right, and
	// only, tool here, not assert.Equal.
	if loc := got.OccurredAt.Location(); loc != time.UTC {
		t.Errorf("Pending()[0].Event.(domain.ConversationStarted).OccurredAt.Location() = %v, want %v", loc, time.UTC)
	}

	assert.NoErr(t, store.Apply(ctx, event, seq), "Apply")

	view, err := store.Get(ctx, event.ConversationID)
	assert.NoErr(t, err, "Get")
	if loc := view.Thread.Messages[0].PostedAt.Location(); loc != time.UTC {
		t.Errorf("Get().Thread.Messages[0].PostedAt.Location() = %v, want %v", loc, time.UTC)
	}
}
