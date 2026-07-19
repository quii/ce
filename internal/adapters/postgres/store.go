package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quii/ce/internal/domain"
)

// Store is the Postgres-backed out.EventStore, out.Outbox and
// out.Projection - one struct backing all three, since they share a
// single pool/schema, mirroring internal/adapters/memory/event_store.go's
// EventStore doing double duty for out.EventStore and out.Outbox.
type Store struct {
	pool    *pgxpool.Pool
	queries *Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: New(pool)}
}

func recipientsToStrings(recipients domain.Recipients) []string {
	raw := make([]string, len(recipients))
	for i, r := range recipients {
		raw[i] = string(r)
	}
	return raw
}

func stringsToRecipients(raw []string) domain.Recipients {
	recipients := make(domain.Recipients, len(raw))
	for i, r := range raw {
		recipients[i] = domain.ParticipantID(r)
	}
	return recipients
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toEnqueueOutboxEntryParams(seq domain.Sequence, event domain.ConversationStarted) EnqueueOutboxEntryParams {
	return EnqueueOutboxEntryParams{
		Sequence:       int64(seq),
		ConversationID: string(event.ConversationID),
		ThreadID:       string(event.ThreadID),
		MessageID:      string(event.MessageID),
		Creator:        string(event.Creator),
		ResourceUrl:    string(event.ResourceURL),
		ThreadTitle:    string(event.ThreadTitle),
		Author:         string(event.Author),
		Recipients:     recipientsToStrings(event.Recipients),
		MessageText:    string(event.MessageText),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	}
}
