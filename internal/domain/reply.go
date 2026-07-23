package domain

import "time"

// MessagePosted is the event raised whenever a message lands on a thread -
// the opening message a conversation is started with, and every reply
// after it. There is no distinction between the two at the event level
// (rule 2 of "conversation event split"/docs/adr/0029-fine-grained-events.md):
// "a message was posted to a thread" is one cohesive fact, with one
// subject (the message) and attributes (author, text, timestamp), so it
// stays a single event type rather than a near-identical one for each of
// its two producers. It carries no ThreadTitle/Recipients - see rule 5 of
// "reply to a thread": posting a message never alters those.
type MessagePosted struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	MessageID      MessageID
	Author         ParticipantID
	MessageText    MessageText
	OccurredAt     time.Time
}

// ReplyParams is the raw, not-yet-validated input for replying to a
// thread - see rule 1 of "reply to a thread". Author and Message are
// pointers so ValidateReply can distinguish "field omitted" from "field
// present but empty", the same posture as StartConversationParams.
type ReplyParams struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	MessageID      MessageID
	Author         *string
	Message        *string
	OccurredAt     time.Time
}

// ValidateReply enforces rule 1 of "reply to a thread" - required fields,
// checkable with no I/O at all (rule 4), independently of whether the
// conversation or thread it targets even exists. The MessagePosted it
// returns isn't yet authorized to be appended - AuthorizeReply still has
// to check it against the thread's actual current state.
func ValidateReply(params ReplyParams) (MessagePosted, error) {
	if params.Author == nil {
		return MessagePosted{}, ErrAuthorRequired
	}
	if params.Message == nil {
		return MessagePosted{}, ErrMessageRequired
	}

	return MessagePosted{
		ConversationID: params.ConversationID,
		ThreadID:       params.ThreadID,
		MessageID:      params.MessageID,
		Author:         ParticipantID(*params.Author),
		MessageText:    MessageText(*params.Message),
		OccurredAt:     params.OccurredAt,
	}, nil
}

// AuthorizeReply applies rules 2-3 of "reply to a thread" against a
// thread's current state, once it's been looked up: the reply's thread
// has to belong to the given conversation (rule 2), and its author has to
// already be one of the thread's frozen participants (rule 3).
func AuthorizeReply(view ConversationView, reply MessagePosted) error {
	if reply.ThreadID != view.Thread.ID {
		return ErrThreadNotFound
	}
	if !view.Thread.HasParticipant(reply.Author) {
		return ErrReplyForbidden
	}

	return nil
}
