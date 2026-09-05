package domain

import "errors"

// ValidationError is every rejection StartConversation can produce - a
// single type so a handler can translate any of them to 400 with one
// errors.As check rather than enumerating each sentinel, see
// docs/adr/0011-domain-errors-stay-domain-errors.md.
type ValidationError struct {
	msg string
}

func (e ValidationError) Error() string {
	return e.msg
}

// NewValidationError lets a driver translate a rejection it can only see
// as wire-level detail (an HTTP 400 body, say) back into the same domain
// error type StartConversation itself returns - see
// docs/adr/0011-domain-errors-stay-domain-errors.md.
func NewValidationError(msg string) ValidationError {
	return ValidationError{msg: msg}
}

var (
	ErrResourceURLRequired   = ValidationError{msg: "resourceUrl is required"}
	ErrThreadTitleRequired   = ValidationError{msg: "threadTitle is required"}
	ErrAuthorRequired        = ValidationError{msg: "author is required"}
	ErrMessageRequired       = ValidationError{msg: "message is required"}
	ErrRecipientsRequired    = ValidationError{msg: "recipients is required"}
	ErrDuplicateRecipient    = ValidationError{msg: "recipients must not contain a duplicate id"}
	ErrAuthorIsRecipient     = ValidationError{msg: "author must not also appear in recipients"}
	ErrParticipantIDRequired = ValidationError{msg: "participantId is required"}
)

// ErrConversationNotFound and ErrProjectionNotCaughtUp are the two ways a
// read can fail to hand back a conversation - a caller of GetConversation
// distinguishes "there's genuinely nothing here" from "the write hasn't
// been projected yet" (rules 8-9), never an HTTP status code.
var (
	ErrConversationNotFound  = errors.New("conversation not found")
	ErrProjectionNotCaughtUp = errors.New("projection has not caught up to the requested sequence")
)

// ErrThreadNotFound and ErrReplyForbidden are rules 2-3 of "reply to a
// thread": a thread that doesn't exist (or doesn't belong to the given
// conversation) is not found, and a reply from someone outside the
// thread's frozen participant set is forbidden - never an HTTP status
// code, see docs/adr/0011-domain-errors-stay-domain-errors.md.
var (
	ErrThreadNotFound = errors.New("thread not found")
	ErrReplyForbidden = errors.New("author is not a participant of this thread")
)
