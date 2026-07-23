-- +goose Up
-- One migration describing the final schema directly - nothing has
-- shipped yet, so there's no historical data to carry forward and no
-- reason to stack this on top of the incremental migrations that only
-- ever existed to get here (see "simplify event storage").
--
-- conversation_events/conversation_outbox narrow to a handful of
-- always-populated columns shared by every event type, plus a jsonb
-- payload holding just the fields specific to that event_type - see
-- docs/adr/0029-fine-grained-events.md (which commits us to adding more
-- event types over time) and docs/adr/0026-sql-spec-first-with-sqlc.md.
CREATE TABLE conversation_events (
    sequence        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type      TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    payload         JSONB NOT NULL
);

CREATE TABLE conversation_outbox (
    sequence        BIGINT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    payload         JSONB NOT NULL,
    done_at         TIMESTAMPTZ
);

CREATE INDEX conversation_outbox_pending_idx ON conversation_outbox (sequence) WHERE done_at IS NULL;

-- A single-row table: the projection's checkpoint, "applied up to sequence
-- N" - see docs/write-path.md. The boolean primary key defaulting to
-- (and only ever being) true is a standard Postgres idiom for enforcing
-- exactly one row.
CREATE TABLE projection_checkpoint (
    id       BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    sequence BIGINT NOT NULL
);

INSERT INTO projection_checkpoint (sequence) VALUES (0);

-- A single merged read model, back to one row per conversation -
-- splitting it into a separate thread_projection was premature: unlike
-- the event log, a projection is a disposable, rebuildable cache, so it
-- doesn't need to anticipate the same multiplicity the write side does.
-- thread_id/thread_title/participants stay nullable, since
-- ConversationCreated is still applied before its ThreadStarted companion
-- within one batch - Get() treats a NULL thread_id as not-found, the
-- same invariant proven by the contract test, checked against a plain
-- column instead of relying on an INNER JOIN's natural behaviour.
CREATE TABLE conversation_projection (
    id           TEXT PRIMARY KEY,
    resource_url TEXT NOT NULL,
    thread_id    TEXT,
    thread_title TEXT,
    participants TEXT[]
);

-- A thread's messages, one row per message, in append order via the
-- global event sequence each was projected from - see
-- docs/write-path.md and rule 7 of "reply to a thread". Unaffected by
-- this story: messages are genuinely one-to-many and already generic
-- over "a message was posted", whichever event produced it.
CREATE TABLE conversation_projection_messages (
    conversation_id TEXT NOT NULL REFERENCES conversation_projection (id),
    sequence        BIGINT NOT NULL,
    author          TEXT NOT NULL,
    message_text    TEXT NOT NULL,
    posted_at       TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (conversation_id, sequence)
);

-- +goose Down
DROP TABLE conversation_projection_messages;
DROP TABLE conversation_projection;
DROP TABLE projection_checkpoint;
DROP TABLE conversation_outbox;
DROP TABLE conversation_events;
