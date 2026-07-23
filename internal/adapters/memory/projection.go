package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type Projection struct {
	mu            sync.Mutex
	conversations map[domain.ConversationID]domain.ConversationView
	checkpoint    domain.Sequence
}

func NewProjection() *Projection {
	return &Projection{conversations: make(map[domain.ConversationID]domain.ConversationView)}
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
			if err := p.applyThreadStarted(e); err != nil {
				return err
			}
		case domain.MessagePosted:
			if err := p.applyMessagePosted(e); err != nil {
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

func (p *Projection) applyThreadStarted(event domain.ThreadStarted) error {
	view, ok := p.conversations[event.ConversationID]
	if !ok {
		return fmt.Errorf("cannot apply thread started for unknown conversation %q", event.ConversationID)
	}

	view.Thread = domain.ThreadView{
		ID:           event.ThreadID,
		Title:        event.ThreadTitle,
		Participants: event.Participants(),
	}
	p.conversations[event.ConversationID] = view

	return nil
}

func (p *Projection) applyMessagePosted(event domain.MessagePosted) error {
	view, ok := p.conversations[event.ConversationID]
	if !ok {
		return fmt.Errorf("cannot apply message for unknown conversation %q", event.ConversationID)
	}

	view.Thread.Messages = append(view.Thread.Messages, domain.MessageView{
		Author:   event.Author,
		Text:     event.MessageText,
		PostedAt: event.OccurredAt,
	})
	p.conversations[event.ConversationID] = view

	return nil
}

func (p *Projection) Get(_ context.Context, id domain.ConversationID) (domain.ConversationView, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// A ConversationCreated on its own (ThreadStarted not yet applied)
	// leaves an entry with a zero-value Thread - not a state any story's
	// rules give a representation for, so it isn't "found" yet either.
	view, ok := p.conversations[id]
	if !ok || view.Thread.ID == "" {
		return domain.ConversationView{}, domain.ErrConversationNotFound
	}

	return view, nil
}

func (p *Projection) Checkpoint(_ context.Context) (domain.Sequence, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.checkpoint, nil
}
