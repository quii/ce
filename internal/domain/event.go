package domain

// Event is every kind of event this codebase's write path can carry
// through the event store, outbox and projection (out.EventStore,
// out.Outbox, out.Projection) - see
// docs/adr/0019-event-sourcing-transactional-outbox.md.
//
// Each variant models one domain fact, not the shape of whichever use
// case happened to raise it - docs/adr/0029-fine-grained-events.md.
// Starting a conversation raises ConversationCreated, ThreadStarted and
// MessagePosted together, atomically, in one write; replying to a thread
// raises a MessagePosted on its own.
//
// It's a sealed interface: isEvent is unexported, so only the three
// variants below (all in this package) can implement it. That makes
// "exactly these variants, from this package" a compile-time property -
// there's no representable "neither" or "several at once" the way a
// struct of nillable pointer fields would allow, and nothing downstream
// needs a broader event-bus abstraction than that.
//
// TypeName names which variant a given Event is - "ConversationCreated",
// "ThreadStarted", "MessagePosted" - the one place that naming lives, so a
// caller that needs it (e.g. "list a conversation's events" rule 5) never
// has to re-derive it with its own type switch.
type Event interface {
	isEvent()
	TypeName() string
}

func (ConversationCreated) isEvent() {}
func (ThreadStarted) isEvent()       {}
func (MessagePosted) isEvent()       {}

func (ConversationCreated) TypeName() string { return "ConversationCreated" }
func (ThreadStarted) TypeName() string       { return "ThreadStarted" }
func (MessagePosted) TypeName() string       { return "MessagePosted" }

// EventRecord pairs an event with the sequence it was appended at -
// Sequence is assigned by the store when an event is appended
// (docs/adr/0019-event-sourcing-transactional-outbox.md), not carried by
// the event itself, so a caller listing a conversation's full history
// needs the pairing, not just the bare event.
type EventRecord struct {
	Sequence Sequence
	Event    Event
}
