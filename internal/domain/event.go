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
type Event interface {
	isEvent()
}

func (ConversationCreated) isEvent() {}
func (ThreadStarted) isEvent()       {}
func (MessagePosted) isEvent()       {}
