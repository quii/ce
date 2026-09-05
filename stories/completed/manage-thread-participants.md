# Manage thread participants

As a calling application, I can add or remove a participant from a specific thread, so that I can control who can see and reply to that thread.

## Rules

1. Participant membership is scoped to an individual thread. Adding or removing a participant from one thread never changes the participants of another thread in the same conversation.
2. A participant relationship is addressed by its opaque, client-generated ID using `PUT` or `DELETE` on `/conversations/{conversationId}/threads/{threadId}/participants/{participantId}`. `PUT` adds the participant and `DELETE` removes them; neither operation has a request body or bulk variant.
3. Any client may add or remove a participant. The operations do not take or inspect an acting participant identity, and do not require the caller to be a current participant.
4. If the conversation does not exist, the thread does not exist, or the thread belongs to a different conversation, either operation returns `404 Not Found`.
5. Adding an absent participant or removing a present participant appends exactly one membership event. It returns `202 Accepted` with a `Location` header for `/conversations/{conversationId}?after=<sequence>`, and the changed membership is visible once the projection catches up.
6. Adding an already-present participant or removing an already-absent participant is an idempotent no-op. It returns `204 No Content` immediately, appends no event, advances no sequence, and has no `Location` header.
7. Participants control whole-thread visibility and reply permission, not message-level access. Adding a participant gives them access to the thread's complete existing message history after projection catch-up. Removing a participant prevents them from seeing the thread in participant-filtered results and from replying, but does not delete or alter its messages.
8. The original author and the final remaining participant may be removed. A thread with no participants remains readable through the unfiltered conversation endpoint but cannot receive replies until a participant is added.

## Scenarios

```gherkin
Feature: Manage thread participants

  Scenario: Add a participant to one thread of a conversation
    Given a conversation has thread "thread-1" with participants "alice" and "bob"
    And the conversation has another thread "thread-2" with participant "dave"
    When participant "carol" is added to "thread-1"
    Then the write is accepted with a projection-catch-up location
    And, once the projection has caught up, "thread-1" has participants "alice", "bob", and "carol"
    And "thread-2" still has participant "dave"

  Scenario: A newly added participant sees the complete history and can reply
    Given a thread has participant "alice" and messages posted by "alice"
    When participant "bob" is added to the thread
    Then, once the projection has caught up, conversations fetched for "bob" include the thread and all its existing messages
    And "bob" can reply to the thread

  Scenario: Removing a participant preserves messages but revokes thread access
    Given a thread has participants "alice" and "bob"
    And "bob" has posted a reply to the thread
    When participant "bob" is removed from the thread
    Then, once the projection has caught up, the thread has participant "alice" but not "bob"
    And the thread still contains "bob"'s reply
    And conversations fetched for "bob" do not include the thread
    And "bob" cannot reply to the thread
    And the unfiltered conversation read still includes the thread and its messages

  Scenario: A non-participant changes a thread's membership
    Given a thread has participant "alice"
    When client "not-a-participant" adds participant "bob" to the thread
    Then the write is accepted
    And, once the projection has caught up, the thread has participants "alice" and "bob"

  Scenario: The original author and final participant can be removed
    Given a thread has only its original author "alice" as a participant
    When participant "alice" is removed from the thread
    Then, once the projection has caught up, the thread has no participants
    And nobody can reply to the thread
    And the thread remains in the unfiltered conversation read

  Scenario: Repeating an add is an immediate no-op
    Given a thread has participant "alice"
    When participant "alice" is added to the thread again
    Then the response is 204 No Content
    And no event is appended
    And no projection-catch-up location is returned

  Scenario: Repeating a removal is an immediate no-op
    Given a thread has no participant "alice"
    When participant "alice" is removed from the thread
    Then the response is 204 No Content
    And no event is appended
    And no projection-catch-up location is returned

  Scenario: Adding or removing on a nonexistent or mismatched thread is not found
    Given no conversation exists with id "missing-conversation"
    When participant "alice" is added to a thread in conversation "missing-conversation"
    Then the request is rejected with 404 Not Found
    When participant "alice" is removed from a thread belonging to another conversation
    Then the request is rejected with 404 Not Found

```

## Specification

`specifications.ManageThreadParticipantsSpecification`
(`specifications/manage_thread_participants.go`) exercises all eight rules
through the membership add/remove in-ports, conversation reads, participant
filtered reads, replies, event history, and relay catch-up. It runs as
`TestManageThreadParticipants` with both the in-process driver
(`specifications/inprocess/driver_test.go`) and the real HTTP/Postgres/relay
container driver (`specifications/container/driver_test.go`).

## HTTP contract note

An empty final path segment does not match the standard-library `ServeMux`
route generated from the required `participantId` path parameter, so it cannot
reach the strict OpenAPI handler as a relationship request. The API has no
custom middleware for that unroutable shape; routable participant IDs reach
the use case, whose domain validation runs before the projection lookup.
