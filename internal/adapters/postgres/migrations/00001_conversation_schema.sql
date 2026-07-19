-- +goose Up
CREATE TABLE conversation_events (
    sequence        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    thread_id       TEXT NOT NULL,
    message_id      TEXT NOT NULL,
    creator         TEXT NOT NULL,
    resource_url    TEXT NOT NULL,
    thread_title    TEXT NOT NULL,
    author          TEXT NOT NULL,
    recipients      TEXT[] NOT NULL,
    message_text    TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE conversation_outbox (
    sequence        BIGINT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    thread_id       TEXT NOT NULL,
    message_id      TEXT NOT NULL,
    creator         TEXT NOT NULL,
    resource_url    TEXT NOT NULL,
    thread_title    TEXT NOT NULL,
    author          TEXT NOT NULL,
    recipients      TEXT[] NOT NULL,
    message_text    TEXT NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    done_at         TIMESTAMPTZ
);

CREATE INDEX conversation_outbox_pending_idx ON conversation_outbox (sequence) WHERE done_at IS NULL;

CREATE TABLE conversation_projection (
    id                TEXT PRIMARY KEY,
    resource_url      TEXT NOT NULL,
    thread_id         TEXT NOT NULL,
    thread_title      TEXT NOT NULL,
    recipients        TEXT[] NOT NULL,
    message_author    TEXT NOT NULL,
    message_text      TEXT NOT NULL,
    message_posted_at TIMESTAMPTZ NOT NULL
);

-- A single-row table: the projection's checkpoint, "applied up to sequence
-- N" - see docs/write-path.md. The boolean primary key defaulting to
-- (and only ever being) true is a standard Postgres idiom for enforcing
-- exactly one row.
CREATE TABLE projection_checkpoint (
    id       BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    sequence BIGINT NOT NULL
);

INSERT INTO projection_checkpoint (sequence) VALUES (0);

-- +goose Down
DROP TABLE projection_checkpoint;
DROP TABLE conversation_projection;
DROP TABLE conversation_outbox;
DROP TABLE conversation_events;
