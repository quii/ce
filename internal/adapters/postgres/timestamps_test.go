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
	created := domain.ConversationCreated{
		ConversationID: "conversation-utc",
		Creator:        domain.PlaceholderCreator,
		ResourceURL:    "https://example.com/orders/utc",
		OccurredAt:     occurredAt,
	}
	threadStarted := domain.ThreadStarted{
		ConversationID: "conversation-utc",
		ThreadID:       "thread-utc",
		ThreadTitle:    "UTC check",
		Author:         "user-1",
		Recipients:     domain.Recipients{},
		OccurredAt:     occurredAt,
	}
	messagePosted := domain.MessagePosted{
		ConversationID: "conversation-utc",
		ThreadID:       "thread-utc",
		MessageID:      "message-utc",
		Author:         "user-1",
		MessageText:    "does this come back as UTC?",
		OccurredAt:     occurredAt,
	}

	_, err = store.Append(ctx, created, threadStarted, messagePosted)
	assert.NoErr(t, err, "Append")

	pending, err := store.Pending(ctx)
	assert.NoErr(t, err, "Pending")
	assert.Len(t, pending, 3, "Pending()")

	// *time.Location has unexported internals cmp.Diff can't traverse -
	// pointer identity (what *Location == compares) is the right, and
	// only, tool here, not assert.Equal. Checked on all three events,
	// not just the last, since each is inserted via its own query.
	created, ok := pending[0].Event.(domain.ConversationCreated)
	if !ok {
		t.Fatalf("Pending()[0].Event = %#v, want a domain.ConversationCreated", pending[0].Event)
	}
	if loc := created.OccurredAt.Location(); loc != time.UTC {
		t.Errorf("Pending()[0].Event.(domain.ConversationCreated).OccurredAt.Location() = %v, want %v", loc, time.UTC)
	}

	gotThreadStarted, ok := pending[1].Event.(domain.ThreadStarted)
	if !ok {
		t.Fatalf("Pending()[1].Event = %#v, want a domain.ThreadStarted", pending[1].Event)
	}
	if loc := gotThreadStarted.OccurredAt.Location(); loc != time.UTC {
		t.Errorf("Pending()[1].Event.(domain.ThreadStarted).OccurredAt.Location() = %v, want %v", loc, time.UTC)
	}

	got, ok := pending[2].Event.(domain.MessagePosted)
	if !ok {
		t.Fatalf("Pending()[2].Event = %#v, want a domain.MessagePosted", pending[2].Event)
	}
	if loc := got.OccurredAt.Location(); loc != time.UTC {
		t.Errorf("Pending()[2].Event.(domain.MessagePosted).OccurredAt.Location() = %v, want %v", loc, time.UTC)
	}

	assert.NoErr(t, store.Apply(ctx, pending...), "Apply")

	view, err := store.Get(ctx, created.ConversationID)
	assert.NoErr(t, err, "Get")
	loc := view.Threads[0].Messages[0].PostedAt.Location()
	assert.True(t, loc == time.UTC, "Get().Threads[0].Messages[0].PostedAt.Location() = %v, want %v", loc, time.UTC)
}
