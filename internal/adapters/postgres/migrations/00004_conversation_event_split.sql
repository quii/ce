-- +goose Up
-- ConversationStarted is retired in favour of three events raised
-- atomically by StartConversation - ConversationCreated, ThreadStarted and
-- MessagePosted (the latter renamed from, and now reused for, the old
-- ReplyPosted) - see docs/adr/0029-fine-grained-events.md. No historical
-- event data exists to preserve, so the events/outbox tables are dropped
-- and recreated for the new event shape rather than migrated column by
-- column.
DROP TABLE conversation_outbox;
DROP TABLE conversation_events;

CREATE TABLE conversation_events (
    sequence        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type      TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    thread_id       TEXT,
    message_id      TEXT,
    creator         TEXT,
    resource_url    TEXT,
    thread_title    TEXT,
    author          TEXT,
    recipients      TEXT[],
    message_text    TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE conversation_outbox (
    sequence        BIGINT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    thread_id       TEXT,
    message_id      TEXT,
    creator         TEXT,
    resource_url    TEXT,
    thread_title    TEXT,
    author          TEXT,
    recipients      TEXT[],
    message_text    TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL,
    done_at         TIMESTAMPTZ
);

CREATE INDEX conversation_outbox_pending_idx ON conversation_outbox (sequence) WHERE done_at IS NULL;

-- The read side splits the same way the write side did: a conversation's
-- own facts (id, resource url) from its thread's (id, title, participants) -
-- rather than one flat row with thread columns that don't exist yet the
-- instant ConversationCreated alone has been applied, before ThreadStarted
-- follows it. conversation_projection_messages is untouched: it was
-- already generic over "a message was posted", not tied to which event
-- kind produced it.
DROP TABLE conversation_projection_messages;
DROP TABLE conversation_projection;

CREATE TABLE conversation_projection (
    id           TEXT PRIMARY KEY,
    resource_url TEXT NOT NULL
);

CREATE TABLE thread_projection (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversation_projection (id),
    title           TEXT NOT NULL,
    participants    TEXT[] NOT NULL
);

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
DROP TABLE thread_projection;
DROP TABLE conversation_projection;

CREATE TABLE conversation_projection (
    id            TEXT PRIMARY KEY,
    resource_url  TEXT NOT NULL,
    thread_id     TEXT NOT NULL,
    thread_title  TEXT NOT NULL,
    participants  TEXT[] NOT NULL
);

CREATE TABLE conversation_projection_messages (
    conversation_id TEXT NOT NULL REFERENCES conversation_projection (id),
    sequence        BIGINT NOT NULL,
    author          TEXT NOT NULL,
    message_text    TEXT NOT NULL,
    posted_at       TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (conversation_id, sequence)
);

DROP TABLE conversation_outbox;
DROP TABLE conversation_events;

CREATE TABLE conversation_events (
    sequence        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type      TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    thread_id       TEXT NOT NULL,
    message_id      TEXT NOT NULL,
    creator         TEXT,
    resource_url    TEXT,
    thread_title    TEXT,
    author          TEXT NOT NULL,
    recipients      TEXT[],
    message_text    TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE conversation_outbox (
    sequence        BIGINT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    thread_id       TEXT NOT NULL,
    message_id      TEXT NOT NULL,
    creator         TEXT,
    resource_url    TEXT,
    thread_title    TEXT,
    author          TEXT NOT NULL,
    recipients      TEXT[],
    message_text    TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    done_at         TIMESTAMPTZ
);

CREATE INDEX conversation_outbox_pending_idx ON conversation_outbox (sequence) WHERE done_at IS NULL;
