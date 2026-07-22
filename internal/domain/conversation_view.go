package domain

import "time"

// ConversationView is the read model served by GetConversation - rule 10
// of the "start a conversation" story. It's a plain, behaviourless
// projection of past events, not an aggregate with invariants of its own,
// see docs/adr/0006-rich-domain-not-anemic.md.
type ConversationView struct {
	ID          ConversationID
	ResourceURL ResourceURL
	Thread      ThreadView
}

// ThreadView's Participants is the union of the thread's original author
// and its recipients, computed once when ConversationStarted is applied to
// build the projection and frozen from then on - a reply never changes it
// (rules 1-2 of "thread participants"). It has no guaranteed order (rule
// 4): it's a set, not a sequence.
type ThreadView struct {
	ID           ThreadID
	Title        ThreadTitle
	Participants Recipients
	Messages     []MessageView
}

// HasParticipant reports whether id is one of the thread's participants -
// exactly the frozen set it was created with (rule 3 of "reply to a
// thread"/rule 5 of "thread participants"; participation changes are
// deferred to a future story).
func (t ThreadView) HasParticipant(id ParticipantID) bool {
	return t.Participants.Contains(id)
}

type MessageView struct {
	Author   ParticipantID
	Text     MessageText
	PostedAt time.Time
}
