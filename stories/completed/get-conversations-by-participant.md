# Get conversations by participant

As a calling application, I can fetch all conversations a participant is involved in - with each conversation's threads filtered to only those the participant appears in - so that a UI can render exactly what that user can see without receiving threads they have no part in.

## Rules

1. A participant is included in a conversation's result if and only if they appear in the `participants` set of at least one of that conversation's threads. If they appear on no thread, the conversation is omitted entirely.
2. Each returned conversation uses the same shape as `GET /conversations/{id}`, but with its threads filtered: only threads whose `participants` set contains the queried participant ID are included. Threads the participant is not part of are never present in the response.
3. Results are ordered by most-recently-active first: the sort key for each conversation is the timestamp of the latest message posted to any of the participant's visible threads within that conversation. A reply bumps a conversation up the list.
4. The query parameter is `participant_id`, consistent with the `ParticipantID` domain type.
5. Pagination is out of scope for this story - all matching conversations are returned in a single response.
6. Auth is out of scope for this story - no caller-identity scoping is applied, consistent with the rest of the system's current `PlaceholderCreator` stance.

## Scenarios

```gherkin
Feature: Get conversations by participant

  Scenario: Participant on one thread of a multi-thread conversation
    Given a conversation exists with two threads
    And the participant is on thread 1 but not thread 2
    When conversations are fetched for that participant
    Then the conversation is returned with only thread 1 present

  Scenario: Participant on no threads of a conversation
    Given a conversation exists
    And the participant is not on any of its threads
    When conversations are fetched for that participant
    Then that conversation is not returned

  Scenario: Participant appears in multiple conversations
    Given two conversations each have a thread the participant is on
    When conversations are fetched for that participant
    Then both conversations are returned, each with only the participant's thread

  Scenario: Results are ordered by most recently active thread
    Given conversation A has the participant on a thread with a message posted at T+1
    And conversation B has the participant on a thread with a message posted at T+2
    When conversations are fetched for that participant
    Then conversation B appears before conversation A

  Scenario: A reply bumps a conversation up the list
    Given conversation A has the participant on a thread with a message posted at T+2
    And conversation B has the participant on a thread with a message posted at T+1
    And a reply is posted to conversation B's thread at T+3
    When conversations are fetched for that participant
    Then conversation B appears before conversation A

  Scenario: Participant on multiple threads within one conversation
    Given a conversation has two threads both containing the participant
    When conversations are fetched for that participant
    Then the conversation is returned with both threads present
    And the sort key is the latest message across both threads

  Scenario: No conversations exist for participant
    Given no conversations include the participant
    When conversations are fetched for that participant
    Then an empty list is returned
```

## Specification

`specifications.ConversationsByParticipantSpecification` (`specifications/conversations_by_participant.go`) exercises all seven scenarios across both the in-process driver (`TestGetConversationsByParticipant` in `specifications/inprocess/driver_test.go`) and the container driver (`TestGetConversationsByParticipant` in `specifications/container/driver_test.go`).

Per `docs/adr/0022-specifications-and-drivers.md`, the specification is written against `ConversationsByParticipantDriver` - an interface composing `in.ConversationStarter`, `in.ConversationGetter`, `in.ThreadAdder`, `in.ThreadReplier`, `in.ConversationsByParticipantGetter`, and `in.Relay`. Relay is needed because the "reply bumps a conversation up the list" and "add thread" scenarios require a real projected conversation before querying; `drainAndWait` from `specifications/reply_helpers.go` bridges the gap between a write returning a sequence and the projection catching up to it.

The rules are enforced in code as follows:

- Rule 1 (thread-level inclusion): `out.Projection.GetByParticipant` filters at the projection layer - memory: `Projection.GetByParticipant` in `internal/adapters/memory/projection.go` iterates conversations, skips threads where `!thread.HasParticipant(id)`, and omits conversations with no visible threads. Postgres: `ListConversationIDsByParticipant` query (`internal/adapters/postgres/queries/conversation_projection.sql`) uses `participants @> ARRAY[$1::text]` to join only threads the participant is on, then `ListParticipantThreadsForConversation` and `ListParticipantMessagesForConversation` retrieve only the relevant threads and messages.
- Rule 2 (same shape as `GET /conversations/{id}`): the `[]domain.ConversationView` returned by `GetByParticipant` is mapped to `[]Conversation` in `ConversationHandler.GetConversationsByParticipant` (`internal/adapters/httpapi/conversation_handler.go`) using the same `toConversation`/`toThread` helpers already in use for the single-conversation endpoint.
- Rule 3 (ordering by most-recently-active): memory: `GetByParticipant` tracks a `latestMessageAt` map (updated in `applyMessagePosted`) keyed by thread ID and sorts the result by that timestamp descending. Postgres: `ListConversationIDsByParticipant` uses `MAX(cpm.posted_at) FILTER (WHERE ...)` per conversation, ordering by `latest_at DESC`.
- Rule 4 (`participant_id` query parameter): `GetConversationsByParticipantParams.ParticipantId` in the generated `server.gen.go`/`client.gen.go`, bound from `participant_id` by oapi-codegen per the spec in `api/openapi.yaml`.
- Rules 5 & 6 (no pagination, no auth): the use case (`in.NewGetConversationsByParticipantUseCase`, `internal/ports/in/get_conversations_by_participant.go`) returns all matches with no cursor/limit parameter and no caller-identity check, consistent with the rest of the system's `PlaceholderCreator` stance.

New SQL queries (`internal/adapters/postgres/queries/conversation_projection.sql`): `ListConversationIDsByParticipant`, `ListParticipantThreadsForConversation`, `ListParticipantMessagesForConversation` - regenerated via sqlc into `internal/adapters/postgres/conversation_projection.sql.go`. The `GetByParticipant` method on `*Store` lives in its own file `internal/adapters/postgres/projection_by_participant.go` to keep `projection.go` within the 250-line limit (`docs/adr/0004-file-length.md`). Contract tests for `GetByParticipant` live in `internal/adapters/contracttest/projection_by_participant.go`, run against both the memory and Postgres adapters.
