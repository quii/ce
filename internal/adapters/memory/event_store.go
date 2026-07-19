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
	events  []domain.ConversationStarted
	pending map[domain.Sequence]domain.ConversationStarted
}

func NewEventStore() *EventStore {
	return &EventStore{pending: make(map[domain.Sequence]domain.ConversationStarted)}
}

func (s *EventStore) Append(_ context.Context, event domain.ConversationStarted) (domain.Sequence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)
	seq := domain.Sequence(len(s.events))
	s.pending[seq] = event

	return seq, nil
}

func (s *EventStore) Enqueue(_ context.Context, seq domain.Sequence, event domain.ConversationStarted) error {
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
