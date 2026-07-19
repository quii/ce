package memory

import (
	"context"
	"sync"

	"github.com/quii/ce/internal/domain"
)

type Projection struct {
	mu            sync.Mutex
	conversations map[domain.ConversationID]domain.ConversationView
	checkpoint    domain.Sequence
}

func NewProjection() *Projection {
	return &Projection{conversations: make(map[domain.ConversationID]domain.ConversationView)}
}

func (p *Projection) Apply(_ context.Context, event domain.ConversationStarted, seq domain.Sequence) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.conversations[event.ConversationID] = domain.ConversationView{
		ID:          event.ConversationID,
		ResourceURL: event.ResourceURL,
		Thread: domain.ThreadView{
			ID:         event.ThreadID,
			Title:      event.ThreadTitle,
			Recipients: event.Recipients,
			Messages: []domain.MessageView{
				{
					Author:   event.Author,
					Text:     event.MessageText,
					PostedAt: event.OccurredAt,
				},
			},
		},
	}
	p.checkpoint = seq

	return nil
}

func (p *Projection) Get(_ context.Context, id domain.ConversationID) (domain.ConversationView, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	view, ok := p.conversations[id]
	if !ok {
		return domain.ConversationView{}, domain.ErrConversationNotFound
	}

	return view, nil
}

func (p *Projection) Checkpoint(_ context.Context) (domain.Sequence, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.checkpoint, nil
}
