# Simplify event storage: JSONB payload, single conversation projection

As the team building ce, I want `conversation_events`/`conversation_outbox` to store a handful of shared columns plus a JSONB payload instead of one nullable column per event-specific field across every event type, and `conversation_projection`/`thread_projection` merged back into a single table, so that adding a new event type - which `docs/adr/0029-fine-grained-events.md` commits us to doing more of, not less - never means another `ALTER TABLE ... ADD COLUMN` and another round of NULL-column bookkeeping threaded through every insert and read path, and the read side isn't split further than the domain model currently warrants.

## Rules

1. `conversation_events` and `conversation_outbox` both narrow to a handful of always-populated columns - `sequence`, `event_type`, `conversation_id`, `occurred_at` (plus `done_at` on the outbox only) - plus one `jsonb` `payload` column holding just the fields specific to that `event_type`, not the ones already covered by the shared columns.
2. Inserting an event no longer needs a distinct query per event type at the SQL layer: one `InsertEvent`/`EnqueueOutboxEntry` query, parameterized by `event_type` and `payload`, replaces `InsertConversationCreatedEvent`/`InsertThreadStartedEvent`/`InsertMessagePostedEvent` (and their `Enqueue*` counterparts). The Go-level type switch moves from "which columns to populate" to "which small payload struct to marshal" - still one case per event type, but each case is a couple of fields, not a full `pgtype`-mapped params struct.
3. Reading an event back still switches on `event_type` (there's no way around needing to know the target type before unmarshalling), but each case is now `json.Unmarshal(payload, &x)` into a small adapter-local payload type, then assembling the domain event from the shared columns plus the unmarshalled payload - not reading N nullable `pgtype.Text` columns per case.
4. `conversation_projection` and `thread_projection` merge back into a single `conversation_projection` table: `id`, `resource_url`, `thread_id`, `thread_title`, `participants`. `thread_id`/`thread_title`/`participants` stay nullable, since `ConversationCreated` is still applied before its `ThreadStarted` companion within one batch - `Get()` treats a NULL/empty `thread_id` as not-found, the same invariant already proven by the contract test added in "conversation event split," just checked against a plain column instead of relying on an `INNER JOIN`'s natural behaviour.
5. `conversation_projection_messages` is unaffected - it stays its own table. Messages are genuinely one-to-many and already fully generic over "a message was posted, from whichever event produced it"; folding it into a JSONB array on the single projection row would trade a plain `INSERT` for a read-modify-write on every reply, a worse concurrency shape for no simplification benefit.
6. Nothing observable changes - the same "nothing leaks past the adapter boundary" bar "conversation event split" held itself to. The existing black-box specifications, and that story's own lower-level tests, continue to pass unmodified except where they inspect Postgres-specific column shape directly (if any do).
7. No historical-data migration concern, and no reason to carry the migration chain forward either: nothing has shipped, so `internal/adapters/postgres/migrations/00001` through `00004` are squashed into a single fresh `00001_conversation_schema.sql` describing the final schema directly - not another incremental migration stacked on top of four that only ever existed to get here.

## Scenarios

This story has no new caller-visible behaviour, so - like "conversation event split" before it - it isn't driven by a new or extended black-box specification. Its scenarios are lower-level: they prove the storage shape changed without anything leaking out.

```gherkin
Feature: Simplify event storage

  Scenario: An event round-trips through the new shared-columns-plus-payload shape
    Given no prior state
    When a caller starts a conversation
    Then the outbox contains three pending entries, each event's specific fields intact after being written to and read back from the payload column

  Scenario: A conversation still isn't readable until its thread has started
    Given a ConversationCreated event has been applied but not its ThreadStarted companion
    When the conversation is read
    Then it is reported as not found, exactly as before this story

  Scenario: Existing behaviour is unchanged
    Given the existing specifications for "start a conversation", "reply to a thread", and "thread participants"
    Then they all continue to pass without modification
```

## Notes for implementation

- Delete `internal/adapters/postgres/migrations/00001` through `00004` outright and replace them with a single fresh `00001_conversation_schema.sql` that creates the final schema directly: `conversation_events`/`conversation_outbox` (`payload jsonb NOT NULL` - even a payload-less event type gets `{}`, never a NULL payload, so unmarshalling never needs a nil-payload branch), `projection_checkpoint`, the single merged `conversation_projection`, and `conversation_projection_messages`. One migration, not five - there's nothing left for the others to preserve.
- Since this rewrites migration history rather than adding to it, any local Postgres volume needs a clean start rather than an incremental `goose up` - `docker compose down -v` (or equivalent) before bringing the stack back up, so goose's own version-tracking table doesn't disagree with the now-renumbered file on disk. Worth calling out explicitly wherever this gets implemented, since it's easy to miss and surfaces as a confusing goose error rather than an obvious one.
- `internal/adapters/postgres/store.go`'s per-event-type `toInsertXEventParams`/`toEnqueueXOutboxEntryParams` functions collapse into a marshal step per event type (a small adapter-local payload struct per event, e.g. `conversationCreatedPayload{Creator, ResourceURL}`) plus one shared `toInsertEventParams(seq, eventType, conversationID, occurredAt, payload)`-shaped helper.
- `internal/adapters/postgres/outbox.go`'s `toDomainEvent` still switches on `event_type`, but each case does `json.Unmarshal(row.Payload, &payload)` into the matching adapter-local payload type, then builds the domain event from the shared columns plus payload fields - no more `row.Author.String`/`row.ThreadTitle.String`-style `pgtype` unwrapping.
- `internal/adapters/postgres/projection.go`'s `applyConversationCreated`/`applyThreadStarted` collapse into upserts against the single `conversation_projection` table; `Get()` drops the `INNER JOIN`, reads one table, and checks `thread_id` (or equivalent) for NULL/empty to preserve the "not readable until `ThreadStarted`" invariant.
- `internal/adapters/memory` is unaffected in shape - it was never wide-column to begin with, just a Go map of `ConversationView`. Its behaviour must still agree with Postgres's via the contract tests, unchanged.
- `internal/adapters/contracttest/{event_store,outbox,projection}.go` should need zero changes if this refactor holds its "nothing observable" line, since they assert behaviour, not storage shape. If any assertion currently depends on Postgres-specific column access, it needs to move to a Postgres-only test.
- Read `docs/adr/0029-fine-grained-events.md` and `docs/adr/0026-sql-spec-first-with-sqlc.md` before starting - this story's payload-column design is downstream of both: 0029 commits us to more event types over time, and 0026 requires sqlc-generated queries stay the source of truth even for a JSONB column, so no hand-written SQL string sneaks into adapter Go code for the payload path either.

## Specification

This story has no black-box specification of its own - rule 6 is proven by the existing `ConversationSpecification`, `ConversationProjectionSpecification`, and `ReplyToThreadSpecification` continuing to pass unmodified against both drivers (`go tool mage test`), which is exactly the point: nothing observable changed.

The storage reshape itself is proven by:

- Rule 1/2/3 (shared columns plus payload, one `InsertEvent`/`EnqueueOutboxEntry` query, `event_type`-switched unmarshal): `internal/adapters/postgres/migrations/00001_conversation_schema.sql` (the new `conversation_events`/`conversation_outbox` shape), `internal/adapters/postgres/queries/conversation_events.sql`/`conversation_outbox.sql` (the single query per table), and `internal/adapters/postgres/store.go`'s `marshalEvent` (one type switch producing `eventType`/`payload` per event, shared by `event_store.go`'s `appendEvent` and `outbox.go`'s `Enqueue`) plus `outbox.go`'s `toDomainEvent` (one type switch, each case a `json.Unmarshal` into a small adapter-local payload type: `conversationCreatedPayload`, `threadStartedPayload`, `messagePostedPayload`).
- Rule 4 (single merged `conversation_projection`, NULL `thread_id` as not-found): `internal/adapters/postgres/queries/conversation_projection.sql`'s `GetConversationProjection` (`WHERE id = $1 AND thread_id IS NOT NULL`, so both "unknown id" and "thread not started yet" come back as the same `pgx.ErrNoRows` / `domain.ErrConversationNotFound`) and `projection.go`'s `applyThreadStarted` (an `UPDATE` against the single table rather than an insert into a separate `thread_projection`).
- Rule 5 (`conversation_projection_messages` unaffected): unchanged in the migration and in `projection.go`/`conversation_projection.sql`.
- Rule 6 (nothing observable changes, contract tests need zero changes): `internal/adapters/contracttest/event_store.go`/`outbox.go`/`projection.go` are unmodified by this story and pass against both `internal/adapters/memory` and the real Postgres adapter via `internal/adapters/postgres/store_test.go`'s `TestStore_Contract` - including "a conversation isn't readable until its thread has started" (`projection.go`), the same invariant rule 4 above targets, now checked against a plain column instead of an `INNER JOIN`.
- Rule 7 (one squashed migration): `internal/adapters/postgres/migrations/00001_conversation_schema.sql` is the only migration file; the previous `00001`-`00004` are deleted.

`go tool mage test` and `go tool mage lint` both pass clean.
