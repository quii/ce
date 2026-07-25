# List a conversation's events

As a calling application, I can list every event raised for a conversation, so that I can audit its full history - not just its current projected state.

## Rules

1. Listing events targets an existing conversation by ID in the URL (`GET /conversations/{conversationId}/events`); there's no request body.
2. If the conversation doesn't exist, the request is rejected `404 Not Found`. Existence is determined directly against the event store, not a projection: a conversation with no events *is* "no such conversation," since every conversation's very first write is a `ConversationCreated` event - there's no separate existence check to perform.
3. Events are returned in the order they were appended (ascending sequence), oldest first - the full history from conversation creation to the most recent write, with no limit on how many are returned.
4. This is a read straight from the append-only event store, not a projection: an event appears in the list the instant its write commits, before the relay has even run. There's no `202`-pending/checkpoint mechanic the way `GET /conversations/{id}` has - this endpoint only ever responds `200` or `404`.
5. Each event in the response carries its sequence number, its event type name, and when it occurred, plus that type's own fields - one flat shape reused across all three event kinds, with only the fields relevant to that event's type populated: `ConversationCreated` carries `creator`/`resourceUrl`; `ThreadStarted` carries `threadId`/`threadTitle`/`author`/`recipients`; `MessagePosted` carries `threadId`/`messageId`/`author`/`messageText`.
6. No authorization check is applied to who may list a conversation's events - any caller can list any existing conversation's events, regardless of participation in any of its threads. Same deferred-authorization posture as rule 5 of "add a thread to a conversation" and every other read in this codebase.

## Scenarios

```gherkin
Feature: List a conversation's events

  Scenario: Listing events for a freshly started conversation
    Given a conversation has been started about resource "https://example.com/orders/123" with thread title "Order query", author "user-1", recipients ["user-2"], and opening message "Where is my order?"
    When a caller lists the conversation's events
    Then the response shows three events in order: "ConversationCreated", "ThreadStarted", "MessagePosted"
    And the "ConversationCreated" event shows resource "https://example.com/orders/123"
    And the "ThreadStarted" event shows thread title "Order query", author "user-1", and recipients ["user-2"]
    And the "MessagePosted" event shows author "user-1" and message text "Where is my order?"

  Scenario: A reply appears as a new event at the end of the list
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-2" replies to the thread with message "Looking into it"
    And a caller lists the conversation's events
    Then the fourth event is a "MessagePosted" event with author "user-2" and message text "Looking into it"

  Scenario: Adding a thread appears as two new events at the end of the list
    Given a conversation has been started with thread title "Order query"
    When a new thread titled "Delivery date" is added to the conversation
    And a caller lists the conversation's events
    Then the list ends with a "ThreadStarted" event for "Delivery date" followed by a "MessagePosted" event for its opening message

  Scenario: Listing events for a nonexistent conversation is not found
    Given no conversation exists with id "missing-conversation"
    When a caller lists events for conversation "missing-conversation"
    Then the request is rejected with 404 Not Found

  Scenario: Events are visible immediately, without waiting for the projection
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When a caller lists the conversation's events immediately after the write, before the projection has processed it
    Then the response already shows every event from that write - there is no pending or 202 state for this endpoint

  Scenario: Anyone can list events, even someone not participating in any thread
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-99", who is not a participant of any thread, lists the conversation's events
    Then the response shows the full event list
```

## Specification

`specifications.ListConversationEventsSpecification` (`specifications/list_conversation_events.go`) exercises every rule (six scenarios: a freshly started conversation's three events in order, each with its own type-specific fields; a reply appearing as a fourth event; adding a thread appearing as two more events at the end of the list; a nonexistent conversation rejected as not found; events visible immediately, before any relay drain; and a non-participant successfully listing events), run via `TestListConversationEvents` against both the in-process driver (`specifications/inprocess/driver_test.go`) and the container driver (`specifications/container/driver_test.go`).

Per `docs/adr/0022-specifications-and-drivers.md`, the specification is written against `in.EventLister` directly, composed into `EventListDriver` (`specifications/list_conversation_events.go`) - `ThreadAddDriver` and `ThreadReplyDriver` embedded together (both already needed by other specifications) plus `in.EventLister`, since scenarios need a conversation with a reply and an added thread to list a realistic history against. Setup reuses `startThreadAndCatchUp`/`reply`/`drainAndWait` (`specifications/reply_helpers.go`) rather than inventing new helpers.

The rules are enforced in code as follows:

- Rule 1 (targeted by URL, no body): `GET /conversations/{conversationId}/events` (`api/openapi.yaml`) takes only a path parameter; `in.ListConversationEventsCommand` (`internal/ports/in/list_conversation_events.go`) has exactly one field, `ConversationID`, a plain string since it always comes from the URL (`docs/adr/0010-tiny-types.md`'s Command exception).
- Rule 2 (404 derived from an empty result, no separate existence check): `listConversationEventsUseCase.ListConversationEvents` (`internal/ports/in/list_conversation_events.go`) treats a zero-length `out.EventStore.ListByConversation` result as `domain.ErrConversationNotFound` - there's no call to `out.Projection` anywhere in this use case.
- Rule 3 (append order): `out.EventStore.ListByConversation` (`internal/ports/out/event_store.go`) is documented to return ascending-sequence order; `memory.EventStore.ListByConversation` (`internal/adapters/memory/event_store.go`) filters its already-append-ordered internal slice; `postgres.Store.ListByConversation` (`internal/adapters/postgres/event_store.go`) runs `ListConversationEvents ... ORDER BY sequence` (`internal/adapters/postgres/queries/conversation_events.sql`). Both proven by the same new contract-test subtests ("listing events for a conversation returns them in append order", "listing events only returns events belonging to the requested conversation", "listing events for a conversation that has never had an event appended returns an empty list" - `internal/adapters/contracttest/event_store.go`), run against both adapters (`docs/adr/0009-contract-tests.md`).
- Rule 4 (read straight from the event store, no pending/checkpoint mechanic): the use case's only dependency is `out.EventStore` - it has no way to reach `out.Projection` or produce `domain.ErrProjectionNotCaughtUp` even if it wanted to, a structural guarantee rather than just a runtime one. Proven directly by the "events are visible immediately, without waiting for the projection" scenario, which lists events with no `Drain` call at all.
- Rule 5 (flat per-event shape, only the relevant fields populated): `domain.EventRecord` (`internal/domain/event.go`) pairs a `Sequence` with an `Event` - `Sequence` is assigned by the store, not carried by the event itself. `domain.Event` gained a `TypeName() string` method (`"ConversationCreated"`/`"ThreadStarted"`/`"MessagePosted"`) alongside its existing sealed `isEvent()`, so any caller needing the type's name (this story, but also the specification's own assertions) has one place to get it rather than re-deriving it with its own type switch. `ConversationHandler.ListConversationEvents`/`toEvent` (`internal/adapters/httpapi/conversation_handler.go`) type-switches on the record's `Event` to populate only that variant's own fields on the generated `Event` response type - `api/openapi.yaml`'s `Event` schema marks only `sequence`/`type`/`occurredAt` required, every other field optional (and so `omitempty`, per `docs/adr/0024-openapi-spec-first-with-oapi-codegen.md`'s generated-pointer convention), the same posture `StartConversationRequest` already uses for "field genuinely absent" vs "present". `specifications/container`'s `toEventRecord` (`specifications/container/event_mapping.go`) is the reverse mapping the container driver needs to hand the same `domain.EventRecord` shape back to the specification, split into its own file (alongside `conversation_mapping.go`, holding the pre-existing `toConversationView`/`toThreadView`/`parseConversationLocation`) to keep `driver.go` under `docs/adr/0004-file-length.md`'s 250-line limit.
- Rule 6 (no authorization): `ListConversationEvents` never inspects a caller's identity or any thread's participants - proven by the "anyone can list events, even someone not participating in any thread" scenario.

`internal/adapters/postgres/outbox.go`'s `toDomainEvent` was widened from taking the outbox-specific `ListPendingOutboxEntriesRow` type to plain columns (sequence, event type, conversation id, occurred-at, payload), so it's shared between `Outbox.Pending` and the new `EventStore.ListByConversation` instead of duplicating the `event_type` switch/payload-unmarshal per sqlc-generated row type - both `conversation_events` and `conversation_outbox` store the identical shape under different generated Go structs.

`go tool mage test` and `go tool mage lint` both pass clean.
