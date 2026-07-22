# Start a conversation

As a calling application, I can start a conversation about a resource by posting an opening message, so that a thread of discussion exists for that resource which I can retrieve later.

## Rules

1. Starting a conversation requires a resource URL, a title for the first thread, an opening message's text, the message author's ID, and a recipients collection (a set of opaque participant IDs) - all fields must be present in the request. Author and recipients together make up the thread's participants.
2. A missing field is rejected with `400 Bad Request` - this includes `recipients`, which must be present even when empty; a caller has to deliberately supply "no recipients" rather than that being the default of omitting the field. A present-but-empty string (for the string fields) or empty collection (for recipients) is otherwise a valid value and accepted as-is - no content validation (length, format, blankness) is applied to any of them.
3. Recipients is treated as a set: a duplicate ID appearing more than once within it is rejected with `400 Bad Request` rather than silently deduplicated.
4. The author's ID must not also appear in recipients; if it does, the request is rejected with `400 Bad Request`.
5. Starting a conversation creates both the conversation and its first thread in one operation; there is no separate step to create the opening thread.
6. The creator identity recorded against a new conversation is a fixed placeholder for this story, not yet derived from a real caller identity - per-application authorization (deriving identity from a request header, and scoping access to it) is deferred to a follow-up story.
7. A successful start responds `202 Accepted` with a `Location` header for the new conversation, carrying the sequence number of the appended event as an `after` query parameter.
8. `GET /conversations/{id}?after=N` returns pending until the read projection's checkpoint reaches sequence `N`, and the full representation once it has.
9. `GET /conversations/{id}` with no `after` parameter is a plain, unconditional read against current projection state - `200` if present, `404` if not - and is never pending, even if the write behind it hasn't been projected yet.
10. The representation of a conversation includes its id, its resource URL, and its thread (id, title, participants, and messages - each message showing the author's ID, the text, and when it was posted). Superseded by rule 1 of the "thread participants" story: participants is a single field, the union of the original author and recipients, rather than author and recipients being kept as separate fields.

## Scenarios

```gherkin
Feature: Start a conversation

  Scenario: A conversation is started with all required fields
    Given no prior state
    When a caller starts a conversation about resource "https://example.com/orders/123" with thread title "Order query", author "user-1", recipients ["user-2", "user-3"], and opening message "Where is my order?"
    Then the write is accepted
    And, once the projection has caught up, reading the conversation shows resource "https://example.com/orders/123", thread title "Order query", recipients ["user-2", "user-3"], and one message from author "user-1" with text "Where is my order?"

  Scenario: Empty string values are accepted
    Given no prior state
    When a caller starts a conversation with thread title "", author "user-1", recipients [], and opening message ""
    Then the write is accepted
    And, once the projection has caught up, reading the conversation shows an empty thread title and a message with empty text

  Scenario: Empty recipients are accepted
    Given no prior state
    When a caller starts a conversation with author "user-1" and an empty recipients collection
    Then the write is accepted
    And, once the projection has caught up, reading the conversation shows no recipients

  Scenario: A missing resource URL is rejected
    Given no prior state
    When a caller starts a conversation with no resource URL
    Then the request is rejected with 400 Bad Request

  Scenario: A missing thread title is rejected
    Given no prior state
    When a caller starts a conversation with no thread title
    Then the request is rejected with 400 Bad Request

  Scenario: A missing author is rejected
    Given no prior state
    When a caller starts a conversation with no author
    Then the request is rejected with 400 Bad Request

  Scenario: A missing opening message is rejected
    Given no prior state
    When a caller starts a conversation with no opening message text
    Then the request is rejected with 400 Bad Request

  Scenario: A missing recipients field is rejected
    Given no prior state
    When a caller starts a conversation with no recipients field at all
    Then the request is rejected with 400 Bad Request

  Scenario: Duplicate recipient IDs are rejected
    Given no prior state
    When a caller starts a conversation with recipients ["user-2", "user-2"]
    Then the request is rejected with 400 Bad Request

  Scenario: An author who is also listed as a recipient is rejected
    Given no prior state
    When a caller starts a conversation with author "user-1" and recipients ["user-1", "user-2"]
    Then the request is rejected with 400 Bad Request

  Scenario: Reading a conversation before the projection has caught up is pending
    Given a conversation has just been started
    When the conversation is read using the sequence number from the write, before the projection has processed it
    Then the read is reported as pending, not as the conversation's data

  Scenario: Reading a conversation after the projection has caught up returns it
    Given a conversation has just been started
    When the projection has processed the write
    And the conversation is then read using the sequence number from the write
    Then the read returns the full conversation representation

  Scenario: A plain read reflects whatever the projection currently holds
    Given a conversation has just been started but the projection has not yet processed it
    When the conversation is read without specifying a sequence number
    Then the read returns 404, not pending, because the projection genuinely has nothing for it yet
```

## Specification

`specifications.ConversationSpecification` (`specifications/conversation.go`) exercises rules 1-4, 8, and 9 (nine scenarios: five missing-field rejections, duplicate recipients, author-as-recipient, the pending-before-catch-up read, and the plain read against an unprojected write), run via `TestStartConversation` against both the in-process driver (`specifications/inprocess/driver_test.go`) and the container driver (`specifications/container/driver_test.go`).

`specifications.ConversationProjectionSpecification` (`specifications/conversation_projection.go`) exercises rules 5-7 and 10 (four scenarios: all required fields, empty string values, empty recipients, and reading after the projection has caught up), run via `TestStartConversation_Projection` against both drivers too.

Per `docs/adr/0022-specifications-and-drivers.md`, both specifications are written directly against the real in-ports - `in.ConversationStarter` and `in.ConversationGetter` (`internal/ports/in/{start_conversation,get_conversation}.go`) - composed into `ConversationDriver`, with `ConversationProjectionDriver` additionally embedding `in.Relay` so a specification can ask a driver to make sure a write has been processed. The in-process driver's `Drain` is the real relay, called synchronously; the container driver's `Drain` is a no-op - there's no HTTP surface to trigger a real, independently-ticking relay container on demand - so `waitForProjection` (`specifications/conversation_projection.go`) polls `GetConversation` the way any real HTTP client is expected to (`docs/write-path.md`) rather than assuming `Drain` alone was enough: a ticker-based poll, never `time.Sleep` (`docs/adr/0021-no-flaky-tests.md`).

One known characteristic, flagged by review and not yet addressed: `ConversationSpecification`'s "pending before the projection has caught up" scenario asserts `domain.ErrProjectionNotCaughtUp` immediately after `StartConversation` returns, with no wait. Run against the container driver's real, independently-ticking relay (~1s), this has no synchronization guarantee - a narrow, timing-dependent flake is possible if the relay's tick lands in that gap.

The rules are enforced in code as follows:

- Rules 1 & 2 (required fields, empty-string-valid): `domain.StartConversation` (`internal/domain/conversation.go`) checks each of `StartConversationParams`' pointer fields for `nil` and returns the matching sentinel (`domain.ErrResourceURLRequired`, `ErrThreadTitleRequired`, `ErrAuthorRequired`, `ErrMessageRequired`, `ErrRecipientsRequired` - `internal/domain/conversation_errors.go`) - a present-but-empty string or empty slice is never checked further.
- Rule 3 (recipients are a set): `domain.NewRecipients` (`internal/domain/conversation.go`) rejects a duplicate ID with `ErrDuplicateRecipient` at construction time - the one place this is checked.
- Rule 4 (author excluded from recipients): `domain.StartConversation` checks `recipients.Contains(author)` and returns `ErrAuthorIsRecipient`.
- Rule 5 (one atomic operation): `domain.StartConversation` returns a single `ConversationStarted` event carrying the conversation, its first thread, and its opening message together - there is no separate thread- or message-creation step.
- Rule 6 (placeholder creator identity): `domain.StartConversation` records `domain.PlaceholderCreator` on every `ConversationStarted` event; real per-application identity is deferred to a follow-up story.
- Rule 7 (202 + Location): `internal/adapters/httpapi/conversation_handler.go` responds `202 Accepted` with a `Location` header built from the use case's `StartConversationResult` (conversation ID and sequence).
- Rules 8 & 9 (pending vs. plain read): `in.GetConversationCommand.After` (`internal/ports/in/get_conversation.go`) is `nil` for a plain read and non-nil to wait for a specific write; `getConversationUseCase.GetConversation` compares it against `out.Projection.Checkpoint`, returning `domain.ErrProjectionNotCaughtUp` when the checkpoint hasn't reached it, or `domain.ErrConversationNotFound`/the real view otherwise. `internal/ports/in/relay.go`'s `Drain` is gap-aware - it only advances the checkpoint over a contiguous run of sequences, so a later-committed sequence can never make a caller think an earlier, still-unprojected one has landed.
- Rule 10 (representation shape): `domain.ConversationView`/`ThreadView`/`MessageView` (`internal/domain/conversation_view.go`) - superseded by rule 1 of the "thread participants" story: `ThreadView.Participants` is now a single field, the union of the original author and recipients, computed once via `ConversationStarted.Participants()` when the projection applies the event (`internal/adapters/memory/projection.go`, `internal/adapters/postgres/projection.go`).
