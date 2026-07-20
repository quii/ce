# Reply to a thread

As a calling application, I can post a reply to an existing thread, so that further discussion is recorded and retrievable as part of the conversation.

## Rules

1. A reply targets a specific conversation and thread by ID in the URL (`POST /conversations/{conversationId}/threads/{threadId}/messages`) plus, in the body, the replying participant's ID (`author`) and the message `text` - both required, same posture as starting a conversation: missing is rejected with `400 Bad Request`; present-but-empty is valid and not further validated.
2. If the conversation doesn't exist, the thread doesn't exist, or the thread exists but doesn't belong to the given conversation, the request is rejected with `404 Not Found`.
3. The reply's author must already be a participant of the thread - its original author or one of its recipients, exactly the set the thread was created with. That set is frozen for this story; participation changes are deferred to a future story. Anyone else is rejected with `403 Forbidden`.
4. Checks are applied in this order: request-shape validation (400, requires no I/O) - existence (404) - authorship (403). A malformed request targeting a nonexistent thread is rejected 400, not 404.
5. A successful reply appends the message to the thread without altering its title or recipients, and responds `202 Accepted` with a `Location` header pointing at the conversation resource, carrying the sequence number of the appended event as an `after` query parameter - the same shape as starting a conversation.
6. The reply's timestamp is set on the command from the server clock at the point the HTTP handler builds it, not supplied by the caller.
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
