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
	Recipients Recipients
	Messages   []MessageView
}

type MessageView struct {
	Author   ParticipantID
	Text     MessageText
	PostedAt time.Time
}
