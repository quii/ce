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

// ConversationStarted is the genesis event for a conversation: starting a
// conversation creates both the conversation and its first thread, with
// its opening message, in one operation - there is no separate event for
// the thread or the message.
type ConversationStarted struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	MessageID      MessageID
	Creator        CreatorID
	ResourceURL    ResourceURL
	ThreadTitle    ThreadTitle
	Author         ParticipantID
	Recipients     Recipients
	MessageText    MessageText
	OccurredAt     time.Time
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
// story.
func StartConversation(params StartConversationParams) (ConversationStarted, error) {
	if params.ResourceURL == nil {
		return ConversationStarted{}, ErrResourceURLRequired
	}
	if params.ThreadTitle == nil {
		return ConversationStarted{}, ErrThreadTitleRequired
	}
	if params.Author == nil {
		return ConversationStarted{}, ErrAuthorRequired
	}
	if params.Message == nil {
		return ConversationStarted{}, ErrMessageRequired
	}
	if params.Recipients == nil {
		return ConversationStarted{}, ErrRecipientsRequired
	}

	recipients, err := NewRecipients(*params.Recipients)
	if err != nil {
		return ConversationStarted{}, err
	}

	author := ParticipantID(*params.Author)
	if recipients.Contains(author) {
		return ConversationStarted{}, ErrAuthorIsRecipient
	}

	return ConversationStarted{
		ConversationID: params.ConversationID,
		ThreadID:       params.ThreadID,
		MessageID:      params.MessageID,
		Creator:        PlaceholderCreator,
		ResourceURL:    ResourceURL(*params.ResourceURL),
		ThreadTitle:    ThreadTitle(*params.ThreadTitle),
		Author:         author,
		Recipients:     recipients,
		MessageText:    MessageText(*params.Message),
		OccurredAt:     params.OccurredAt,
	}, nil
}
