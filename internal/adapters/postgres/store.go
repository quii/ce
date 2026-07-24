package postgres

import (
	"encoding/json"
	"fmt"
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

// conversationCreatedPayload/threadStartedPayload/messagePostedPayload are
// the small, adapter-local JSON shapes stored in conversation_events'/
// conversation_outbox's payload column - just the fields specific to
// their event_type, not the shared columns (conversation_id, occurred_at)
// already covered by conversation_events/conversation_outbox's own
// columns - see docs/adr/0029-fine-grained-events.md and rule 1 of
// "simplify event storage".
type conversationCreatedPayload struct {
	Creator     string `json:"creator"`
	ResourceURL string `json:"resource_url"`
}

type threadStartedPayload struct {
	ThreadID    string   `json:"thread_id"`
	ThreadTitle string   `json:"thread_title"`
	Author      string   `json:"author"`
	Recipients  []string `json:"recipients"`
}

type messagePostedPayload struct {
	ThreadID    string `json:"thread_id"`
	MessageID   string `json:"message_id"`
	Author      string `json:"author"`
	MessageText string `json:"message_text"`
}

// eventRow is the shared-columns-plus-payload shape both
// conversation_events and conversation_outbox store a domain.Event as -
// see rule 1 of "simplify event storage". marshalEvent builds one from
// any domain.Event, so appendEvent (event_store.go) and Enqueue
// (outbox.go) share a single type switch instead of each having their
// own.
type eventRow struct {
	EventType      string
	ConversationID domain.ConversationID
	OccurredAt     time.Time
	Payload        []byte
}

func marshalEvent(event domain.Event) (eventRow, error) {
	switch e := event.(type) {
	case domain.ConversationCreated:
		payload, err := json.Marshal(conversationCreatedPayload{
			Creator:     string(e.Creator),
			ResourceURL: string(e.ResourceURL),
		})
		if err != nil {
			return eventRow{}, fmt.Errorf("could not marshal ConversationCreated payload: %w", err)
		}
		return eventRow{
			EventType:      eventTypeConversationCreated,
			ConversationID: e.ConversationID,
			OccurredAt:     e.OccurredAt,
			Payload:        payload,
		}, nil
	case domain.ThreadStarted:
		payload, err := json.Marshal(threadStartedPayload{
			ThreadID:    string(e.ThreadID),
			ThreadTitle: string(e.ThreadTitle),
			Author:      string(e.Author),
			Recipients:  recipientsToStrings(e.Recipients),
		})
		if err != nil {
			return eventRow{}, fmt.Errorf("could not marshal ThreadStarted payload: %w", err)
		}
		return eventRow{
			EventType:      eventTypeThreadStarted,
			ConversationID: e.ConversationID,
			OccurredAt:     e.OccurredAt,
			Payload:        payload,
		}, nil
	case domain.MessagePosted:
		payload, err := json.Marshal(messagePostedPayload{
			ThreadID:    string(e.ThreadID),
			MessageID:   string(e.MessageID),
			Author:      string(e.Author),
			MessageText: string(e.MessageText),
		})
		if err != nil {
			return eventRow{}, fmt.Errorf("could not marshal MessagePosted payload: %w", err)
		}
		return eventRow{
			EventType:      eventTypeMessagePosted,
			ConversationID: e.ConversationID,
			OccurredAt:     e.OccurredAt,
			Payload:        payload,
		}, nil
	default:
		return eventRow{}, fmt.Errorf("cannot marshal event of unrecognized type %T", event)
	}
}
