//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/adapters/postgres/postgrestest"
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
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
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
	if err != nil {
		t.Fatalf("Append returned an unexpected error: %v", err)
	}

	pending, err := store.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending returned an unexpected error: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("Pending() = %#v, want exactly one entry", pending)
	}
	if loc := pending[0].Event.OccurredAt.Location(); loc != time.UTC {
		t.Errorf("Pending()[0].Event.OccurredAt.Location() = %v, want %v", loc, time.UTC)
	}

	if err := store.Apply(ctx, event, seq); err != nil {
		t.Fatalf("Apply returned an unexpected error: %v", err)
	}

	view, err := store.Get(ctx, event.ConversationID)
	if err != nil {
		t.Fatalf("Get returned an unexpected error: %v", err)
	}
	if loc := view.Thread.Messages[0].PostedAt.Location(); loc != time.UTC {
		t.Errorf("Get().Thread.Messages[0].PostedAt.Location() = %v, want %v", loc, time.UTC)
	}
}
