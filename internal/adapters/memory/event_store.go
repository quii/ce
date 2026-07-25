package memory

import (
	"context"
	"slices"
	"sync"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// EventStore is an in-memory out.EventStore and out.Outbox. Appending an
// event and enqueuing its outbox row are two separate out-port calls
// (docs/write-path.md), but mirroring internal/adapters/postgres.Store's
// real transactional-outbox behaviour, Append enqueues the event itself -
// the use case's subsequent Enqueue call is always a redundant, harmless
// no-op. This is what keeps the fake and the real adapter agreeing on the
// same contract test, per docs/adr/0009-contract-tests.md.
type EventStore struct {
	mu      sync.Mutex
	records []domain.EventRecord
	pending map[domain.Sequence]domain.Event
}

func NewEventStore() *EventStore {
	return &EventStore{pending: make(map[domain.Sequence]domain.Event)}
}

func (s *EventStore) Append(_ context.Context, events ...domain.Event) (domain.Sequence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var seq domain.Sequence
	for _, event := range events {
		seq = domain.Sequence(len(s.records) + 1)
		s.records = append(s.records, domain.EventRecord{Sequence: seq, Event: event})
		s.pending[seq] = event
	}

	return seq, nil
}

// ListByConversation scans every appended event in order - a single
// unfiltered slice is exactly what a fake backing a single-process test
// needs, unlike internal/adapters/postgres.Store, which can push the
// conversation_id filter down into the query itself.
func (s *EventStore) ListByConversation(_ context.Context, id domain.ConversationID) ([]domain.EventRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var records []domain.EventRecord
	for _, record := range s.records {
		if conversationIDOf(record.Event) == id {
			records = append(records, record)
		}
	}

	return records, nil
}

// conversationIDOf extracts the ConversationID every domain.Event variant
// carries - a type switch rather than an interface method, mirroring
// internal/adapters/postgres/store.go's marshalEvent, since Event's own
// sealed interface (event.go) deliberately exposes nothing beyond isEvent.
func conversationIDOf(event domain.Event) domain.ConversationID {
	switch e := event.(type) {
	case domain.ConversationCreated:
		return e.ConversationID
	case domain.ThreadStarted:
		return e.ConversationID
	case domain.MessagePosted:
		return e.ConversationID
	default:
		return ""
	}
}

func (s *EventStore) Enqueue(_ context.Context, seq domain.Sequence, event domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pending[seq] = event
	return nil
}

func (s *EventStore) Pending(_ context.Context) ([]out.OutboxEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sequences := make([]domain.Sequence, 0, len(s.pending))
	for seq := range s.pending {
		sequences = append(sequences, seq)
	}
	slices.Sort(sequences)

	entries := make([]out.OutboxEntry, len(sequences))
	for i, seq := range sequences {
		entries[i] = out.OutboxEntry{Sequence: seq, Event: s.pending[seq]}
	}

	return entries, nil
}

func (s *EventStore) MarkDone(_ context.Context, seq domain.Sequence) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pending, seq)
	return nil
}
