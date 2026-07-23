package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quii/ce/internal/domain"
)

const (
	eventTypeConversationCreated = "ConversationCreated"
	eventTypeThreadStarted       = "ThreadStarted"
	eventTypeMessagePosted       = "MessagePosted"
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

func toNullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func toInsertConversationCreatedEventParams(event domain.ConversationCreated) InsertConversationCreatedEventParams {
	return InsertConversationCreatedEventParams{
		ConversationID: string(event.ConversationID),
		Creator:        toNullableText(string(event.Creator)),
		ResourceUrl:    toNullableText(string(event.ResourceURL)),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	}
}

func toEnqueueConversationCreatedOutboxEntryParams(seq domain.Sequence, event domain.ConversationCreated) EnqueueConversationCreatedOutboxEntryParams {
	return EnqueueConversationCreatedOutboxEntryParams{
		Sequence:       int64(seq),
		ConversationID: string(event.ConversationID),
		Creator:        toNullableText(string(event.Creator)),
		ResourceUrl:    toNullableText(string(event.ResourceURL)),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	}
}

func toInsertThreadStartedEventParams(event domain.ThreadStarted) InsertThreadStartedEventParams {
	return InsertThreadStartedEventParams{
		ConversationID: string(event.ConversationID),
		ThreadID:       toNullableText(string(event.ThreadID)),
		ThreadTitle:    toNullableText(string(event.ThreadTitle)),
		Author:         toNullableText(string(event.Author)),
		Recipients:     recipientsToStrings(event.Recipients),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	}
}

func toEnqueueThreadStartedOutboxEntryParams(seq domain.Sequence, event domain.ThreadStarted) EnqueueThreadStartedOutboxEntryParams {
	return EnqueueThreadStartedOutboxEntryParams{
		Sequence:       int64(seq),
		ConversationID: string(event.ConversationID),
		ThreadID:       toNullableText(string(event.ThreadID)),
		ThreadTitle:    toNullableText(string(event.ThreadTitle)),
		Author:         toNullableText(string(event.Author)),
		Recipients:     recipientsToStrings(event.Recipients),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	}
}

func toInsertMessagePostedEventParams(event domain.MessagePosted) InsertMessagePostedEventParams {
	return InsertMessagePostedEventParams{
		ConversationID: string(event.ConversationID),
		ThreadID:       toNullableText(string(event.ThreadID)),
		MessageID:      toNullableText(string(event.MessageID)),
		Author:         toNullableText(string(event.Author)),
		MessageText:    toNullableText(string(event.MessageText)),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	}
}

func toEnqueueMessagePostedOutboxEntryParams(seq domain.Sequence, event domain.MessagePosted) EnqueueMessagePostedOutboxEntryParams {
	return EnqueueMessagePostedOutboxEntryParams{
		Sequence:       int64(seq),
		ConversationID: string(event.ConversationID),
		ThreadID:       toNullableText(string(event.ThreadID)),
		MessageID:      toNullableText(string(event.MessageID)),
		Author:         toNullableText(string(event.Author)),
		MessageText:    toNullableText(string(event.MessageText)),
		OccurredAt:     toTimestamptz(event.OccurredAt),
	}
}
