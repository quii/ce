# Conversation event split

As the team building ce, I want conversation-creation modelled as three separate, atomically-raised events instead of one bundled use-case event, so future stories (a new thread on an existing conversation, tags, etc.) can reuse the exact fact they need instead of rebuilding it - without changing anything a caller can observe today.

## Rules

1. `ConversationStarted` is retired. `StartConversation` instead raises three events atomically, in one write: `ConversationCreated` (creator, resource URL), `ThreadStarted` (thread ID, title, author, recipients), and `MessagePosted` (conversation ID, thread ID, message ID, author, text, timestamp).
2. `ReplyPosted` is renamed to `MessagePosted` and reused as-is for the opening message - a reply and an opening message are the same fact ("a message was posted to a thread"), with no distinction at the event level.
3. A single logical write can raise multiple events, committed atomically in one transaction with sequential sequence numbers - `out.EventStore.Append` accepts a batch of events, not one at a time, per `docs/adr/0029-fine-grained-events.md`.
4. Nothing observable changes: `StartConversation`'s command/result shape, its validation rules, `ReplyToThread`, and `GetConversation`'s representation are all unchanged. The existing black-box specifications (`ConversationSpecification`, `ConversationProjectionSpecification`, `ReplyToThreadSpecification`) continue to pass with zero modification - that's the proof nothing leaked out.
5. No historical-data migration concern - free to reshape the Postgres schema outright (per `docs/adr/0029-fine-grained-events.md`), since no prior event shape has been released.

## Scenarios

This story has no new caller-visible behaviour, so - unlike every other story so far - it isn't driven by a new or extended black-box specification against an in-port. Its scenarios are lower-level: they prove the internal event-sourcing mechanism, not something a calling application observes.

```gherkin
Feature: Conversation event split

  Scenario: Starting a conversation raises three events atomically
    Given no prior state
    When a caller starts a conversation
    Then the outbox contains exactly three pending entries - ConversationCreated, ThreadStarted, and MessagePosted, in that order - with sequential sequence numbers assigned in one write

  Scenario: Posting a reply raises a MessagePosted event
    Given a conversation has been started
    When a reply is posted to the thread
    Then the outbox contains a MessagePosted event for that reply

  Scenario: Existing behaviour is unchanged
    Given the existing specifications for "start a conversation", "reply to a thread", and "thread participants"
    Then they all continue to pass without modification
```

## Notes for implementation

- `out.EventStore.Append` should accept a batch of events (e.g. `Append(ctx, events ...domain.Event) (domain.Sequence, error)`, returning the *last* sequence in the batch - sufficient for `StartConversationResult.Sequence`/the `Location` header's `after=N`, since a client waiting for the last event in an atomic write implies the earlier ones already landed too). Both the memory and Postgres adapters need updating; the Postgres adapter's transaction needs to insert N event rows and N outbox rows (sequential sequences) in one commit.
- `out.Projection.Apply` likewise needs to accept the same batch and apply every event in it before advancing the checkpoint once - not once per event, since a partially-applied batch would let a reader observe e.g. `ThreadStarted` without its `MessagePosted` companion.
- This note turned out to be wrong, caught during review: `internal/ports/in/relay.go`'s `Drain` **does** need to change. Draining one row at a time in order, gap-aware, is still correct for detecting the gap itself - but applying each row via its own `Projection.Apply` call, rather than the whole contiguous run in one call, reopens exactly the partially-applied state `Apply`'s batching exists to rule out: a plain, unconditional `GET` could observe a `ThreadStarted` with no `MessagePosted` companion yet, in the gap between two separate `Drain` steps. `Drain` now accumulates the contiguous run into a slice and calls `Projection.Apply` once with all of it, only calling `Outbox.MarkDone` per entry afterward - see `TestRelay_Drain_AppliesAContiguousRunInOneApplyCall` (`internal/ports/in/relay_test.go`).
- The new lower-level tests belong in `internal/ports/in/start_conversation_test.go` (extend or add alongside `TestStartConversation_OnlyNeedsEventStore` - construct the real `memory.NewEventStore()`, call the use case, then inspect `.Pending(ctx)` directly for the three events/sequences) and the shared contract-test suites (`internal/adapters/contracttest/event_store.go`/`outbox.go`/`projection.go`, extended to cover a multi-event `Append` call against both the fake and the real Postgres adapter).
- `stories/completed/start-a-conversation.md`'s rule 5 ("there is no separate event for the thread or message") needs updating to describe the new three-event shape, the same way rule 10 was updated for "thread participants" - see `docs/adr/0029-fine-grained-events.md`'s Consequences section.
- Read `docs/adr/0029-fine-grained-events.md` before starting - it's the design record this story is a direct application of.

## Specification

This story has no black-box specification of its own - rule 4 is proven by the existing `ConversationSpecification`, `ConversationProjectionSpecification`, and `ReplyToThreadSpecification` continuing to pass unmodified against both drivers, which is exactly the point: nothing observable changed.

The event-sourcing mechanism itself is proven by lower-level tests:

- Rule 1 (three atomic events): `TestStartConversation_RaisesThreeEventsAtomically` (`internal/ports/in/start_conversation_test.go`) - starts a conversation against a real `memory.NewEventStore()` and inspects `Pending()` directly, asserting `ConversationCreated`, `ThreadStarted`, and `MessagePosted` land with sequential sequences `[1,2,3]` in one `Append` call, and that `StartConversationResult.Sequence` is the last of the three.
- Rule 2 (`ReplyPosted` renamed to `MessagePosted`, reused for the opening message): `internal/domain/reply.go`/`conversation.go` - `ValidateReply` and `domain.StartConversation` both produce `domain.MessagePosted`; the existing reply and start-conversation domain tests exercise both call sites of the one type.
- Rule 3 (`Append`/`Apply` accept a batch): `internal/adapters/contracttest/event_store.go` ("appending a batch of events returns the sequence of the last event in the batch"), `outbox.go` ("appending a batch of events assigns sequential sequences and enqueues all of them, in order, with one call"), and `projection.go` ("applying a batch of events makes the conversation readable and advances the checkpoint once", "a conversation isn't readable until its thread has started", "applying zero entries is a no-op that leaves the checkpoint untouched") - each run against both `internal/adapters/memory` and `internal/adapters/postgres` via `internal/adapters/postgres/store_test.go`'s `TestStore_Contract`, so the two adapters are proven to agree, not just individually correct.
- Rule 5 (no migration concern): `internal/adapters/postgres/migrations/00004_conversation_event_split.sql` drops and recreates `conversation_events`/`conversation_outbox` (wide, nullable-per-event-type columns, discriminated by `event_type` - not JSONB) and splits `conversation_projection` into `conversation_projection` (id, resource URL) plus a new `thread_projection` (id, conversation ID, title, participants), rather than migrating column-by-column.

One behaviour surfaced during review, not anticipated by the rules as written: a `ConversationCreated` applied without its `ThreadStarted` companion (a state that can only arise mid-batch, never durably, since both land in the same transaction) must not make the conversation readable - `Get()` treats a zero-value `Thread` the same as not-found, matching Postgres's `INNER JOIN`-based behaviour naturally rather than needing an explicit check there. Both adapters agree on this via the contract test named above, and `internal/ports/in/relay_test.go`'s gap-arithmetic tests were adjusted to stop relying on a bare `ConversationCreated` being independently readable, since that was never a real state, just a convenient stand-in event for testing checkpoint/gap bookkeeping.

A second, more consequential gap surfaced during review, once the write side genuinely committed batches atomically: `internal/ports/in/relay.go`'s `Drain` was still applying the outbox one row at a time, which meant a plain, unconditional `GET` could observe a `ThreadStarted` with no `MessagePosted` companion yet, in the gap between two separate `Drain` steps - the exact partially-applied state `Projection.Apply`'s own batching was introduced to rule out. `Drain` now accumulates a contiguous run of pending entries and applies the whole run in one `Projection.Apply` call, proven by `TestRelay_Drain_AppliesAContiguousRunInOneApplyCall` (`internal/ports/in/relay_test.go`), which wraps the real `memory.Projection` fake to record how many entries each `Apply` call received.
