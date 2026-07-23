# Thread participants

As a calling application, I can read a conversation's thread and see all of its participants as a single set, instead of reconstructing that from separate author/recipients fields myself.

## Rules

1. A thread's representation exposes a single `participants` field - the union of the thread's original author and its recipients - replacing the previously separate author/recipients distinction in the read model. This supersedes rule 10 of the completed "start a conversation" story: the representation now shows `participants` in place of a standalone `recipients` field, and no longer exposes the original author as a distinct thing to reconcile against it.
2. `participants` is computed once, when the `ThreadStarted` event is applied to build the projection (`ConversationStarted` at the time this story was written - split into `ConversationCreated`/`ThreadStarted`/`MessagePosted` by the later "conversation event split" story), and stays fixed thereafter - posting a reply never changes it (the same "frozen participant set" guarantee reply-to-a-thread's rule 3 already relies on).
3. The write side is unaffected: `POST /conversations` still takes `recipients` as an input field, and the `ThreadStarted` event still records author and recipients separately in the durable event log - only the read-side representation changes shape. Nothing about starting a conversation or replying to a thread changes from a caller's point of view on the write path.
4. `participants` has no guaranteed order - it's a set, not a sequence.
5. The reply-to-a-thread authorization check (only a thread's participants may reply) is defined directly as membership in this same `participants` set - not recomputed independently from author and recipients.

## Scenarios

```gherkin
Feature: Thread participants

  Scenario: Reading a conversation shows participants as the union of author and recipients
    Given a conversation has been started with author "user-1" and recipients ["user-2", "user-3"]
    When the conversation is read
    Then the thread's participants are exactly "user-1", "user-2", and "user-3" (in any order)

  Scenario: Participants are unchanged after a reply is posted
    Given a conversation has been started with author "user-1" and recipients ["user-2"]
    And "user-2" has replied to the thread
    When the conversation is read
    Then the thread's participants are still exactly "user-1" and "user-2"

  Scenario: A conversation started with no recipients shows participants as just the author
    Given a conversation has been started with author "user-1" and an empty recipients collection
    When the conversation is read
    Then the thread's participants are exactly "user-1"
```

## Specification

`specifications.ConversationProjectionSpecification` (`specifications/conversation_projection.go`) exercises all 5 rules - the existing "start a conversation" scenarios now also assert the participants union directly, plus two scenarios added for this story: participants unchanged after a reply, and participants-is-just-the-author when recipients is empty - run via `TestStartConversation_Projection` against both the in-process driver (`specifications/inprocess/driver_test.go`) and the container driver (`specifications/container/driver_test.go`).

`ConversationProjectionDriver` was widened to also embed `in.ThreadReplier` (alongside the `ConversationDriver`/`in.Relay` it already had), since the "unchanged after a reply" scenario needs a real reply posted against an already-projected thread - the same shape `ThreadReplyDriver` (`specifications/reply.go`) already needed for its own specification. `assertRecipients` was renamed to `assertParticipants` and changed from positional equality to membership (`assert.Len` + `assert.Contains` per element), matching rule 4 - two participant sets with the same members in a different order are the same value as far as this story's rules go.

The rules are enforced in code as follows:

- Rule 1 (single field, the union): `domain.ThreadView.Participants` (`internal/domain/conversation_view.go`) replaces the old `Author`/`Recipients` fields. The union itself is computed by `ThreadStarted.Participants()` (`internal/domain/conversation.go` - `ConversationStarted.Participants()` at the time this story was written, moved by the later "conversation event split" story onto the event that now actually carries the thread's author/recipients), the one place it's derived - both adapters call it rather than each re-deriving the union independently.
- Rule 2 (computed once, frozen): `applyThreadStarted` (`internal/adapters/memory/projection.go`, `internal/adapters/postgres/projection.go`) sets `Participants` only when a `ThreadStarted` event is applied; `applyMessagePosted` in both files never touches it.
- Rule 3 (write side unaffected): `StartConversationCommand`/`StartConversationParams`/the `ThreadStarted` event still carry `Recipients` exactly as before this story. The Postgres migration (`internal/adapters/postgres/migrations/00003_thread_participants.sql`) only changes the read-side `conversation_projection` table (replacing its `thread_author`/`recipients` columns with one `participants` column, data-preserving in both directions) - `conversation_events`/`conversation_outbox` are untouched.
- Rule 4 (no guaranteed order): `assertParticipants` checks membership, not positional equality - see above.
- Rule 5 (authorization via participants): `ThreadView.HasParticipant` (`internal/domain/conversation_view.go`) is `t.Participants.Contains(id)`, used unchanged by `domain.AuthorizeReply` (`internal/domain/reply.go`) - reply-to-a-thread's authorization rule didn't need to change, only what it reads from.

Rule 10 of `stories/completed/start-a-conversation.md`, and that story's corresponding "rules enforced in code" note, were updated to point at this story rather than describe the now-superseded separate-fields shape.
