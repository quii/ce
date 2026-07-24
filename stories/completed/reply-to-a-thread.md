# Reply to a thread

As a calling application, I can post a reply to an existing thread, so that further discussion is recorded and retrievable as part of the conversation.

## Rules

1. A reply targets a specific conversation and thread by ID in the URL (`POST /conversations/{conversationId}/threads/{threadId}/messages`) plus, in the body, the replying participant's ID (`author`) and the message `text` - both required, same posture as starting a conversation: missing is rejected with `400 Bad Request`; present-but-empty is valid and not further validated.
2. If the conversation doesn't exist, the thread doesn't exist, or the thread exists but doesn't belong to the given conversation, the request is rejected with `404 Not Found`.
3. The reply's author must already be a participant of the thread - its original author or one of its recipients, exactly the set the thread was created with. That set is frozen for this story; participation changes are deferred to a future story. Anyone else is rejected with `403 Forbidden`.
4. Checks are applied in this order: request-shape validation (400, requires no I/O) - existence (404) - authorship (403). A malformed request targeting a nonexistent thread is rejected 400, not 404.
5. A successful reply appends the message to the thread without altering its title or recipients, and responds `202 Accepted` with a `Location` header pointing at the conversation resource, carrying the sequence number of the appended event as an `after` query parameter - the same shape as starting a conversation.
6. The reply's timestamp is set from the server clock (the same injected `Clock` out-port "start a conversation" uses for its opening message) when the use case builds the event, not supplied by the caller.
7. A thread's messages are ordered by append order: the opening message first, then replies in the order they were posted.
8. The pending/plain-read semantics already established for `GET /conversations/{id}` (rules 8-9 of "Start a conversation") apply unchanged to reads that include a reply - a reply is just another write the same checkpoint mechanism has to catch up to.

## Scenarios

```gherkin
Feature: Reply to a thread

  Scenario: The thread's original author replies
    Given a conversation has been started about resource "https://example.com/orders/123" with thread title "Order query", author "user-1", recipients ["user-2"], and opening message "Where is my order?"
    When "user-1" replies to the thread with message "Let me know when you can"
    Then the write is accepted
    And, once the projection has caught up, reading the conversation shows two messages: "Where is my order?" from "user-1" then "Let me know when you can" from "user-1"

  Scenario: A recipient replies
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-2" replies to the thread with message "Looking into it"
    Then the write is accepted
    And, once the projection has caught up, reading the conversation shows the reply from "user-2"

  Scenario: Someone outside the thread's participants is forbidden from replying
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-3" replies to the thread with message "Can I help?"
    Then the request is rejected with 403 Forbidden

  Scenario: Replying to a nonexistent conversation is not found
    Given no conversation exists with id "missing-conversation"
    When "user-1" replies to a thread in conversation "missing-conversation"
    Then the request is rejected with 404 Not Found

  Scenario: Replying with a thread ID that doesn't belong to the given conversation is not found
    Given a conversation "conversation-a" has been started with thread "thread-a"
    And a different conversation "conversation-b" has been started with thread "thread-b"
    When "user-1" replies to thread "thread-b" under conversation "conversation-a"
    Then the request is rejected with 404 Not Found

  Scenario: A missing author is rejected
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When a reply is posted to the thread with no author
    Then the request is rejected with 400 Bad Request

  Scenario: Missing message text is rejected
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-1" replies to the thread with no message text
    Then the request is rejected with 400 Bad Request

  Scenario: Empty string message text is accepted
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-1" replies to the thread with message text ""
    Then the write is accepted
    And, once the projection has caught up, reading the conversation shows a reply with empty text

  Scenario: A malformed request to a nonexistent thread is rejected as a bad request, not a not-found
    Given no conversation exists with id "missing-conversation"
    When a reply with no author is posted to a thread in conversation "missing-conversation"
    Then the request is rejected with 400 Bad Request

  Scenario: Replies land in posting order
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-1" replies with message "first reply"
    And, once that write has been projected, "user-2" replies with message "second reply"
    Then, once the projection has caught up, reading the conversation shows messages in order: the opening message, then "first reply", then "second reply"

  Scenario: A reply is pending until the projection catches up
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    When "user-1" replies to the thread with message "Let me know when you can"
    And the conversation is read using the sequence number from the reply, before the projection has processed it
    Then the read is reported as pending, not as the conversation's data
    And, once the projection has processed the reply, reading the conversation again returns the full representation including the reply
```

## Specification

`specifications.ReplyToThreadSpecification` (`specifications/reply.go`) exercises all 8 rules (eleven scenarios: the original author replying, a recipient replying, a non-participant forbidden, a nonexistent conversation, a thread ID that doesn't belong to the given conversation, a missing author, missing message text, empty-string message text, a malformed request against a nonexistent thread, replies landing in posting order, and the pending-before-catch-up read), run via `TestReplyToThread` against both the in-process driver (`specifications/inprocess/driver_test.go`) and the container driver (`specifications/container/driver_test.go`).

Per `docs/adr/0022-specifications-and-drivers.md`, the specification is written directly against the real in-ports, composed into `ThreadReplyDriver` (`specifications/reply.go`) - `ConversationDriver` (to start the conversation a reply targets) embedded with `in.ThreadReplier` and `in.Relay`: every scenario needs a real, already-projected conversation to reply against, since rules 2-3 look the thread's current participants up via the projection before a reply can be authorized, so this driver needs `Relay` even though the plainer `ConversationDriver` used by `start-a-conversation`'s specifications doesn't. `specifications/reply_helpers.go` holds the shared scaffolding: `startThreadAndCatchUp` (start a conversation and drain it so a reply has something real to target), `reply` (a thin wrapper over `driver.ReplyToThread`), `drainAndWait`, and the assertion helpers `assertReplyRejected` (checks both the error's type and its exact message, so a driver falling back to a generic message instead of propagating the real one fails loudly), `assertReplyNotFound` (accepts either `domain.ErrConversationNotFound` or `domain.ErrThreadNotFound`, since a container-driven 404 has no machine-readable way to distinguish which one it was), and `assertMessagesInOrder`.

This story is the second event kind through the write path first built for "start a conversation" - see `docs/adr/0019-event-sourcing-transactional-outbox.md`. `domain.Event` (`internal/domain/event.go`) became a sealed interface (`ConversationStarted` | `ReplyPosted` at the time this story was written, exactly one variant since only types in the `domain` package can implement the unexported `isEvent()` marker method - the later "conversation event split" story split `ConversationStarted` further and renamed `ReplyPosted` to `MessagePosted`, reused as-is for the opening message, so the sealed interface is now `ConversationCreated` | `ThreadStarted` | `MessagePosted`) rather than the single hard-coded event type it was before, threaded through `out.EventStore`, `out.Outbox`, and `out.Projection` and both adapters (`internal/adapters/memory/*`, `internal/adapters/postgres/*`). The Postgres migration (`internal/adapters/postgres/migrations/00002_reply_to_thread.sql`) splits the single-message `conversation_projection` row into `conversation_projection` (the thread's header - id, title, author, recipients) plus an append-ordered `conversation_projection_messages` table, since a thread can now have more than one message.

The rules are enforced in code as follows:

- Rule 1 (required fields): `domain.ValidateReply` (`internal/domain/reply.go`) checks `Author`/`Message` for `nil` and returns `domain.ErrAuthorRequired`/`domain.ErrMessageRequired` - the same sentinels "start a conversation" uses, reused rather than duplicated. It's pure and does no I/O, which is what makes rule 4's ordering possible.
- Rules 2 & 3 (existence, authorship): `domain.AuthorizeReply` (`internal/domain/reply.go`), given the conversation's current `ConversationView` (fetched by the use case), looks the reply's thread up via `view.Thread(reply.ThreadID)` (`domain.ErrThreadNotFound` if not found - updated by "add a thread to a conversation" from a direct `view.Thread.ID` comparison, once `Thread` became a list) and checks `!thread.HasParticipant(reply.Author)` (`domain.ErrReplyForbidden`). `ThreadView.HasParticipant` (`internal/domain/conversation_view.go`) is the one place "is this ID a participant" is checked - the thread's frozen author or one of its recipients.
- Rule 4 (check ordering): `replyToThreadUseCase.ReplyToThread` (`internal/ports/in/reply_to_thread.go`) calls `domain.ValidateReply` first (no I/O), then `Projection.Get` (existence - a nonexistent conversation surfaces `domain.ErrConversationNotFound` here), then `domain.AuthorizeReply` (authorship), then `Events.Append`. `ConversationHandler.ReplyToThread` (`internal/adapters/httpapi/conversation_handler.go`) maps `domain.ValidationError` to 400, `domain.ErrConversationNotFound`/`ErrThreadNotFound` to 404, and `domain.ErrReplyForbidden` to 403, checked in that order.
- Rule 5 (202 + Location, unaltered title/recipients): `MessagePosted` (`internal/domain/reply.go` - `ReplyPosted` at the time this story was written, renamed by the later "conversation event split" story) carries no `ThreadTitle`/`Recipients` fields at all, so there's nothing for `applyMessagePosted` (`internal/adapters/memory/projection.go`, `internal/adapters/postgres/projection.go`) to overwrite - it only appends a message row. `ConversationHandler.ReplyToThread` builds the same `/conversations/{id}?after=N` Location string `StartConversation` does.
- Rule 6 (server-set timestamp): `replyToThreadUseCase.ReplyToThread` sets `OccurredAt: uc.deps.Clock.Now()` when building `domain.ReplyParams` - the same injected `out.Clock` dependency `startConversationUseCase` uses, not a caller-supplied value anywhere in `ReplyToThreadCommand`.
- Rule 7 (append order): messages are stored in a table keyed by `(conversation_id, sequence)` (`conversation_projection_messages`) and read back via `ORDER BY sequence` (`ListConversationProjectionMessages`, `internal/adapters/postgres/queries/conversation_projection.sql`); the in-memory projection (`internal/adapters/memory/projection.go`) achieves the same by appending to a slice in the order events are applied.
- Rule 8 (pending/plain-read semantics): unchanged from "start a conversation" - `GetConversation`'s handling of `in.GetConversationCommand.After` doesn't distinguish which event kind produced the sequence it's waiting for, so replies fall under rules 8-9 of that story for free.
