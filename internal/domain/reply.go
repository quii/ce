package domain

import "time"

// ReplyPosted is the event appended when a reply lands on an existing
// thread - see rule 5 of "reply to a thread": it appends a message
// without altering the thread's title or recipients, so it carries none
// of those fields at all.
type ReplyPosted struct {
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
// conversation or thread it targets even exists. The ReplyPosted it
// returns isn't yet authorized to be appended - AuthorizeReply still has
// to check it against the thread's actual current state.
func ValidateReply(params ReplyParams) (ReplyPosted, error) {
	if params.Author == nil {
		return ReplyPosted{}, ErrAuthorRequired
	}
	if params.Message == nil {
		return ReplyPosted{}, ErrMessageRequired
	}

	return ReplyPosted{
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
func AuthorizeReply(view ConversationView, reply ReplyPosted) error {
	if reply.ThreadID != view.Thread.ID {
		return ErrThreadNotFound
	}
	if !view.Thread.HasParticipant(reply.Author) {
		return ErrReplyForbidden
	}

	return nil
}
