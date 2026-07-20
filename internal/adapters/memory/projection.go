package memory

import (
	"context"
	"fmt"
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

func (p *Projection) Apply(_ context.Context, event domain.Event, seq domain.Sequence) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch e := event.(type) {
	case domain.ConversationStarted:
		p.applyConversationStarted(e)
	case domain.ReplyPosted:
		if err := p.applyReplyPosted(e); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot apply event of unrecognized type %T", event)
	}
	p.checkpoint = seq

	return nil
}

func (p *Projection) applyConversationStarted(event domain.ConversationStarted) {
	p.conversations[event.ConversationID] = domain.ConversationView{
		ID:          event.ConversationID,
		ResourceURL: event.ResourceURL,
		Thread: domain.ThreadView{
			ID:         event.ThreadID,
			Title:      event.ThreadTitle,
			Author:     event.Author,
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
}

func (p *Projection) applyReplyPosted(event domain.ReplyPosted) error {
	view, ok := p.conversations[event.ConversationID]
	if !ok {
		return fmt.Errorf("cannot apply reply for unknown conversation %q", event.ConversationID)
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
