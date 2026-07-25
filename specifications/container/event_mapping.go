package container

import (
	"github.com/quii/ce/internal/adapters/apiclient"
	"github.com/quii/ce/internal/domain"
)

// toEventRecord maps a generated apiclient.Event (the flat, type-tagged
// wire shape - "list a conversation's events" rule 5) back into a
// domain.EventRecord, the same domain.Event variant the in-process driver
// hands the specification directly - split out of driver.go purely to keep
// that file under docs/adr/0004-file-length.md's limit.
func toEventRecord(e apiclient.Event) domain.EventRecord {
	record := domain.EventRecord{Sequence: domain.Sequence(e.Sequence)}

	switch e.Type {
	case "ConversationCreated":
		record.Event = domain.ConversationCreated{
			Creator:     domain.CreatorID(stringOrEmpty(e.Creator)),
			ResourceURL: domain.ResourceURL(stringOrEmpty(e.ResourceUrl)),
			OccurredAt:  e.OccurredAt,
		}
	case "ThreadStarted":
		var recipients domain.Recipients
		if e.Recipients != nil {
			recipients = make(domain.Recipients, len(*e.Recipients))
			for i, r := range *e.Recipients {
				recipients[i] = domain.ParticipantID(r)
			}
		}
		record.Event = domain.ThreadStarted{
			ThreadID:    domain.ThreadID(stringOrEmpty(e.ThreadId)),
			ThreadTitle: domain.ThreadTitle(stringOrEmpty(e.ThreadTitle)),
			Author:      domain.ParticipantID(stringOrEmpty(e.Author)),
			Recipients:  recipients,
			OccurredAt:  e.OccurredAt,
		}
	case "MessagePosted":
		record.Event = domain.MessagePosted{
			ThreadID:    domain.ThreadID(stringOrEmpty(e.ThreadId)),
			MessageID:   domain.MessageID(stringOrEmpty(e.MessageId)),
			Author:      domain.ParticipantID(stringOrEmpty(e.Author)),
			MessageText: domain.MessageText(stringOrEmpty(e.MessageText)),
			OccurredAt:  e.OccurredAt,
		}
	}

	return record
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
