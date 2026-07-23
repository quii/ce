package domain

import "time"

type ConversationID string

type ThreadID string

type MessageID string

type ParticipantID string

type ResourceURL string

type ThreadTitle string

type MessageText string

// CreatorID identifies the calling application that started a
// conversation - distinct from a ParticipantID, which identifies a
// participant within it. PlaceholderCreator is a fixed stand-in per rule
// 6 of the "start a conversation" story: deriving a real caller identity
// (and scoping access to it) is deferred to a follow-up story.
type CreatorID string

const PlaceholderCreator CreatorID = "placeholder-creator"

// Sequence is the monotonic position of an appended event in the event
// store - the "after=N" cursor a caller polls a projection's checkpoint
// against, see docs/write-path.md.
type Sequence int64

// Recipients is a set of participant IDs: constructing one is the one
// place duplicate recipients get rejected, so nothing downstream needs to
// re-check it.
type Recipients []ParticipantID

func NewRecipients(raw []string) (Recipients, error) {
	seen := make(map[ParticipantID]struct{}, len(raw))
	recipients := make(Recipients, 0, len(raw))

	for _, r := range raw {
		id := ParticipantID(r)
		if _, ok := seen[id]; ok {
			return nil, ErrDuplicateRecipient
		}
		seen[id] = struct{}{}
		recipients = append(recipients, id)
	}

	return recipients, nil
}

func (r Recipients) Contains(id ParticipantID) bool {
	for _, recipient := range r {
		if recipient == id {
			return true
		}
	}
	return false
}

// ConversationCreated is raised once, the moment a conversation is
// created - see docs/adr/0029-fine-grained-events.md. It used to be
// bundled together with the thread it started and its opening message
// into one ConversationStarted event, but "a conversation was created" is
// a single, cohesive fact on its own (test 2 of that ADR): a conversation
// has many threads in the domain model, so the thread it happens to be
// created with here doesn't belong on this event.
type ConversationCreated struct {
	ConversationID ConversationID
	Creator        CreatorID
	ResourceURL    ResourceURL
	OccurredAt     time.Time
}

// ThreadStarted is raised whenever a new thread begins on a conversation -
// today only ever as part of StartConversation, but kept as its own event
// (rather than folded into ConversationCreated) because a conversation
// already has many threads in the domain model
// (docs/adr/0029-fine-grained-events.md's test 2), independently of how
// many callers currently produce this fact.
type ThreadStarted struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	ThreadTitle    ThreadTitle
	Author         ParticipantID
	Recipients     Recipients
	OccurredAt     time.Time
}

// Participants returns the union of the event's author and recipients -
// the thread's participant set, computed once from this event and frozen
// from then on (rules 1-2 of "thread participants"). It has no guaranteed
// order: it's a set, not a sequence.
func (e ThreadStarted) Participants() Recipients {
	participants := make(Recipients, 0, len(e.Recipients)+1)
	participants = append(participants, e.Author)
	participants = append(participants, e.Recipients...)
	return participants
}

// StartConversationParams is the raw, not-yet-validated input for starting
// a conversation. The string/slice fields are pointers so a caller can
// distinguish "field omitted" from "field present but empty" - see rule 2
// of the "start a conversation" story.
type StartConversationParams struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	MessageID      MessageID
	ResourceURL    *string
	ThreadTitle    *string
	Author         *string
	Recipients     *[]string
	Message        *string
	OccurredAt     time.Time
}

// StartConversation is the single place every rule for starting a
// conversation is enforced - see rules 1-4 of the "start a conversation"
// story. It raises three events atomically, in this order: a
// ConversationCreated, a ThreadStarted for its first thread, and a
// MessagePosted for the opening message - see
// docs/adr/0029-fine-grained-events.md. The caller (out.EventStore.Append)
// is responsible for committing all three in one write.
func StartConversation(params StartConversationParams) ([]Event, error) {
	if params.ResourceURL == nil {
		return nil, ErrResourceURLRequired
	}
	if params.ThreadTitle == nil {
		return nil, ErrThreadTitleRequired
	}
	if params.Author == nil {
		return nil, ErrAuthorRequired
	}
	if params.Message == nil {
		return nil, ErrMessageRequired
	}
	if params.Recipients == nil {
		return nil, ErrRecipientsRequired
	}

	recipients, err := NewRecipients(*params.Recipients)
	if err != nil {
		return nil, err
	}

	author := ParticipantID(*params.Author)
	if recipients.Contains(author) {
		return nil, ErrAuthorIsRecipient
	}

	return []Event{
		ConversationCreated{
			ConversationID: params.ConversationID,
			Creator:        PlaceholderCreator,
			ResourceURL:    ResourceURL(*params.ResourceURL),
			OccurredAt:     params.OccurredAt,
		},
		ThreadStarted{
			ConversationID: params.ConversationID,
			ThreadID:       params.ThreadID,
			ThreadTitle:    ThreadTitle(*params.ThreadTitle),
			Author:         author,
			Recipients:     recipients,
			OccurredAt:     params.OccurredAt,
		},
		MessagePosted{
			ConversationID: params.ConversationID,
			ThreadID:       params.ThreadID,
			MessageID:      params.MessageID,
			Author:         author,
			MessageText:    MessageText(*params.Message),
			OccurredAt:     params.OccurredAt,
		},
	}, nil
}
