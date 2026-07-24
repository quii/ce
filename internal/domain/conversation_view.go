package domain

import "time"

// ConversationView is the read model served by GetConversation - rule 10
// of the "start a conversation" story. It's a plain, behaviourless
// projection of past events, not an aggregate with invariants of its own,
// see docs/adr/0006-rich-domain-not-anemic.md. Threads is a list, not a
// single value, since a conversation is no longer assumed to have exactly
// one thread (rule 9 of "add a thread to a conversation") - in creation
// order (rule 10): the original thread first, then each added thread in
// the order its ThreadStarted event was appended.
type ConversationView struct {
	ID          ConversationID
	ResourceURL ResourceURL
	Threads     []ThreadView
}

// Thread returns the thread with the given id, and whether it was found -
// the one place a caller looks a specific thread up out of the list, used
// by AuthorizeReply (rule 2 of "reply to a thread"). Adding a thread's own
// existence check (rule 4 of "add a thread to a conversation") only needs
// to know whether the conversation exists at all, so it goes through
// out.Projection.Exists instead of this method.
func (c ConversationView) Thread(id ThreadID) (ThreadView, bool) {
	for _, t := range c.Threads {
		if t.ID == id {
			return t, true
		}
	}
	return ThreadView{}, false
}

// ThreadView's Participants is the union of the thread's original author
// and its recipients, computed once when ThreadStarted is applied to
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
