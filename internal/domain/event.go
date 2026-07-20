package domain

// Event is every kind of event this codebase's write path can carry
// through the event store, outbox and projection (out.EventStore,
// out.Outbox, out.Projection) - exactly the two this project has so far,
// see docs/adr/0019-event-sourcing-transactional-outbox.md.
//
// It's a sealed interface: isEvent is unexported, so only ConversationStarted
// and ReplyPosted (both in this package) can implement it. That makes
// "exactly one variant, from this package" a compile-time property -
// there's no representable "neither" or "both" the way a struct of two
// nillable pointer fields would allow, and nothing downstream needs a
// broader event-bus abstraction than that.
type Event interface {
	isEvent()
}

func (ConversationStarted) isEvent() {}
func (ReplyPosted) isEvent()         {}
