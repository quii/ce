package memory

import (
	"fmt"

	"github.com/quii/ce/internal/domain"
)

func (p *Projection) applyParticipantAdded(event domain.ParticipantAdded) error {
	return p.changeParticipant(event.ConversationID, event.ThreadID, event.ParticipantID, true)
}

func (p *Projection) applyParticipantRemoved(event domain.ParticipantRemoved) error {
	return p.changeParticipant(event.ConversationID, event.ThreadID, event.ParticipantID, false)
}

func (p *Projection) changeParticipant(conversationID domain.ConversationID, threadID domain.ThreadID, participantID domain.ParticipantID, add bool) error {
	view, ok := p.conversations[conversationID]
	if !ok {
		return fmt.Errorf("cannot change participant for unknown conversation %q", conversationID)
	}
	for i, thread := range view.Threads {
		if thread.ID != threadID {
			continue
		}
		if add {
			view.Threads[i].Participants = append(thread.Participants, participantID)
		} else {
			var participants domain.Recipients
			for _, p := range thread.Participants {
				if p != participantID {
					participants = append(participants, p)
				}
			}
			view.Threads[i].Participants = participants
		}
		p.conversations[conversationID] = view
		return nil
	}
	return fmt.Errorf("cannot change participant for unknown thread %q on conversation %q", threadID, conversationID)
}
