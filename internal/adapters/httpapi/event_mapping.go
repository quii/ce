package httpapi

import (
	"time"

	"github.com/quii/ce/internal/domain"
)

func toEvent(record domain.EventRecord) Event {
	event := Event{Sequence: int64(record.Sequence), Type: record.Event.TypeName(), OccurredAt: eventOccurredAt(record.Event)}
	switch e := record.Event.(type) {
	case domain.ConversationCreated:
		creator, resourceURL := string(e.Creator), string(e.ResourceURL)
		event.Creator, event.ResourceUrl = &creator, &resourceURL
	case domain.ThreadStarted:
		threadID, threadTitle, author := string(e.ThreadID), string(e.ThreadTitle), string(e.Author)
		recipients := make([]string, len(e.Recipients))
		for i, r := range e.Recipients {
			recipients[i] = string(r)
		}
		event.ThreadId, event.ThreadTitle, event.Author, event.Recipients = &threadID, &threadTitle, &author, &recipients
	case domain.MessagePosted:
		threadID, messageID, author, messageText := string(e.ThreadID), string(e.MessageID), string(e.Author), string(e.MessageText)
		event.ThreadId, event.MessageId, event.Author, event.MessageText = &threadID, &messageID, &author, &messageText
	case domain.ParticipantAdded:
		threadID, participantID := string(e.ThreadID), string(e.ParticipantID)
		event.ThreadId, event.ParticipantId = &threadID, &participantID
	case domain.ParticipantRemoved:
		threadID, participantID := string(e.ThreadID), string(e.ParticipantID)
		event.ThreadId, event.ParticipantId = &threadID, &participantID
	}
	return event
}

func eventOccurredAt(event domain.Event) time.Time {
	switch e := event.(type) {
	case domain.ConversationCreated:
		return e.OccurredAt
	case domain.ThreadStarted:
		return e.OccurredAt
	case domain.MessagePosted:
		return e.OccurredAt
	case domain.ParticipantAdded:
		return e.OccurredAt
	case domain.ParticipantRemoved:
		return e.OccurredAt
	default:
		return time.Time{}
	}
}
