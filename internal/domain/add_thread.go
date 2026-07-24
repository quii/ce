package domain

import "time"

// AddThreadParams is the raw, not-yet-validated input for adding a new
// thread to an existing conversation - see rules 1-2 of "add a thread to a
// conversation". The same required-field/present-but-empty posture as
// StartConversationParams, minus ResourceURL, since the conversation
// already exists - and, field for field, exactly threadParams'
// (conversation.go), which is what lets AddThread convert straight to it
// below rather than rebuilding it field by field.
type AddThreadParams struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	MessageID      MessageID
	ThreadTitle    *string
	Author         *string
	Recipients     *[]string
	Message        *string
	OccurredAt     time.Time
}

// AddThread is the single place every rule for adding a thread to an
// existing conversation is enforced - rules 1-2, 6-7 of "add a thread to a
// conversation". It raises the same two events StartConversation's thread
// half already raises - a ThreadStarted for the new thread and a
// MessagePosted for its opening message - reused rather than duplicated
// (rule 6), and no ConversationCreated, since the conversation already
// exists. Whether the conversation itself exists (rule 4) needs a
// projection lookup, which is the use case's job, not this pure
// function's - see docs/adr/0029-fine-grained-events.md.
func AddThread(params AddThreadParams) ([]Event, error) {
	threadStarted, messagePosted, err := newThread(threadParams(params))
	if err != nil {
		return nil, err
	}

	return []Event{threadStarted, messagePosted}, nil
}
