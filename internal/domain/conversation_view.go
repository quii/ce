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

type ThreadView struct {
	ID         ThreadID
	Title      ThreadTitle
	Author     ParticipantID
	Recipients Recipients
	Messages   []MessageView
}

// HasParticipant reports whether id is one of the thread's participants -
// its original author or one of its recipients, exactly the set it was
// created with (rule 3 of "reply to a thread"; participation changes are
// deferred to a future story).
func (t ThreadView) HasParticipant(id ParticipantID) bool {
	return t.Author == id || t.Recipients.Contains(id)
}

type MessageView struct {
	Author   ParticipantID
	Text     MessageText
	PostedAt time.Time
}
