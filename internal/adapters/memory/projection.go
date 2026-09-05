package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type Projection struct {
	mu            sync.Mutex
	conversations map[domain.ConversationID]domain.ConversationView
	// threadSequences records the sequence each conversation's threads
	// were started at, so Threads can be kept ordered by that sequence
	// (rule 10 of "add a thread to a conversation") rather than by
	// whatever order applyThreadStarted happened to be called in - the
	// same ORDER BY sequence the Postgres adapter reads back with, not an
	// incidental property of apply-call order.
	threadSequences map[domain.ConversationID]map[domain.ThreadID]domain.Sequence
	// latestMessageAt records the latest message timestamp per thread,
	// used to sort conversations by most-recently-active for
	// GetByParticipant (rule 3 of "get conversations by participant").
	latestMessageAt map[domain.ThreadID]time.Time
	checkpoint      domain.Sequence
}

func NewProjection() *Projection {
	return &Projection{
		conversations:   make(map[domain.ConversationID]domain.ConversationView),
		threadSequences: make(map[domain.ConversationID]map[domain.ThreadID]domain.Sequence),
		latestMessageAt: make(map[domain.ThreadID]time.Time),
	}
}

// Apply applies every entry in the batch, in order, before advancing the
// checkpoint once - to the last entry's sequence - rather than once per
// entry, so a reader can never observe the checkpoint having moved past a
// partially-applied batch (docs/adr/0029-fine-grained-events.md).
func (p *Projection) Apply(_ context.Context, entries ...out.OutboxEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, entry := range entries {
		switch e := entry.Event.(type) {
		case domain.ConversationCreated:
			p.applyConversationCreated(e)
		case domain.ThreadStarted:
			if err := p.applyThreadStarted(e, entry.Sequence); err != nil {
				return err
			}
		case domain.MessagePosted:
			if err := p.applyMessagePosted(e); err != nil {
				return err
			}
		case domain.ParticipantAdded:
			if err := p.applyParticipantAdded(e); err != nil {
				return err
			}
		case domain.ParticipantRemoved:
			if err := p.applyParticipantRemoved(e); err != nil {
				return err
			}
		default:
			return fmt.Errorf("cannot apply event of unrecognized type %T", entry.Event)
		}
	}

	if len(entries) > 0 {
		p.checkpoint = entries[len(entries)-1].Sequence
	}

	return nil
}

func (p *Projection) applyConversationCreated(event domain.ConversationCreated) {
	p.conversations[event.ConversationID] = domain.ConversationView{
		ID:          event.ConversationID,
		ResourceURL: event.ResourceURL,
	}
}

// applyThreadStarted appends a new ThreadView to the conversation's
// Threads, then re-sorts by the sequence each thread was started at - rule
// 10 of "add a thread to a conversation" - rather than relying on
// applyThreadStarted's own call order, which Projection.Apply's contract
// doesn't actually guarantee matches sequence order for a given
// conversation across separate batches. This mirrors the Postgres
// adapter, which stores the sequence explicitly and orders by it at read
// time (ListConversationProjectionThreads).
func (p *Projection) applyThreadStarted(event domain.ThreadStarted, seq domain.Sequence) error {
	view, ok := p.conversations[event.ConversationID]
	if !ok {
		return fmt.Errorf("cannot apply thread started for unknown conversation %q", event.ConversationID)
	}

	sequences, ok := p.threadSequences[event.ConversationID]
	if !ok {
		sequences = make(map[domain.ThreadID]domain.Sequence)
		p.threadSequences[event.ConversationID] = sequences
	}
	sequences[event.ThreadID] = seq

	view.Threads = append(view.Threads, domain.ThreadView{
		ID:           event.ThreadID,
		Title:        event.ThreadTitle,
		Participants: event.Participants(),
	})
	sort.SliceStable(view.Threads, func(i, j int) bool {
		return sequences[view.Threads[i].ID] < sequences[view.Threads[j].ID]
	})
	p.conversations[event.ConversationID] = view

	return nil
}

func (p *Projection) applyMessagePosted(event domain.MessagePosted) error {
	view, ok := p.conversations[event.ConversationID]
	if !ok {
		return fmt.Errorf("cannot apply message for unknown conversation %q", event.ConversationID)
	}

	for i, thread := range view.Threads {
		if thread.ID != event.ThreadID {
			continue
		}

		view.Threads[i].Messages = append(thread.Messages, domain.MessageView{
			Author:   event.Author,
			Text:     event.MessageText,
			PostedAt: event.OccurredAt,
		})
		p.conversations[event.ConversationID] = view

		if event.OccurredAt.After(p.latestMessageAt[event.ThreadID]) {
			p.latestMessageAt[event.ThreadID] = event.OccurredAt
		}

		return nil
	}

	return fmt.Errorf("cannot apply message for unknown thread %q on conversation %q", event.ThreadID, event.ConversationID)
}

func (p *Projection) Get(_ context.Context, id domain.ConversationID) (domain.ConversationView, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// A ConversationCreated on its own (its first ThreadStarted not yet
	// applied) leaves an entry with no threads at all - not a state any
	// story's rules give a representation for, so it isn't "found" yet
	// either.
	view, ok := p.conversations[id]
	if !ok || len(view.Threads) == 0 {
		return domain.ConversationView{}, domain.ErrConversationNotFound
	}

	return view, nil
}

// GetByParticipant returns all conversations the participant appears in,
// with threads filtered to those the participant is part of, ordered by
// most-recently-active first (rule 3 of "get conversations by participant").
func (p *Projection) GetByParticipant(_ context.Context, id domain.ParticipantID) ([]domain.ConversationView, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	type entry struct {
		view     domain.ConversationView
		latestAt time.Time
	}

	var entries []entry
	for _, conv := range p.conversations {
		var filtered []domain.ThreadView
		var latest time.Time
		for _, thread := range conv.Threads {
			if !thread.HasParticipant(id) {
				continue
			}
			filtered = append(filtered, thread)
			if t := p.latestMessageAt[thread.ID]; t.After(latest) {
				latest = t
			}
		}
		if len(filtered) == 0 {
			continue
		}
		entries = append(entries, entry{
			view: domain.ConversationView{
				ID:          conv.ID,
				ResourceURL: conv.ResourceURL,
				Threads:     filtered,
			},
			latestAt: latest,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].latestAt.After(entries[j].latestAt)
	})

	views := make([]domain.ConversationView, len(entries))
	for i, e := range entries {
		views[i] = e.view
	}
	return views, nil
}

// Exists reports whether a conversation is currently readable - the same
// "has at least one thread" test Get's not-found check uses, without
// building the full ConversationView Get returns.
func (p *Projection) Exists(_ context.Context, id domain.ConversationID) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	view, ok := p.conversations[id]
	return ok && len(view.Threads) > 0, nil
}

func (p *Projection) Checkpoint(_ context.Context) (domain.Sequence, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.checkpoint, nil
}
