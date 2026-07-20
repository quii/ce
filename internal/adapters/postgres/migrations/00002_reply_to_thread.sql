-- +goose Up
ALTER TABLE conversation_events ADD COLUMN event_type TEXT NOT NULL DEFAULT 'ConversationStarted';
ALTER TABLE conversation_events ALTER COLUMN event_type DROP DEFAULT;
ALTER TABLE conversation_events ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE conversation_events ALTER COLUMN resource_url DROP NOT NULL;
ALTER TABLE conversation_events ALTER COLUMN thread_title DROP NOT NULL;
ALTER TABLE conversation_events ALTER COLUMN recipients DROP NOT NULL;

ALTER TABLE conversation_outbox ADD COLUMN event_type TEXT NOT NULL DEFAULT 'ConversationStarted';
ALTER TABLE conversation_outbox ALTER COLUMN event_type DROP DEFAULT;
ALTER TABLE conversation_outbox ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE conversation_outbox ALTER COLUMN resource_url DROP NOT NULL;
ALTER TABLE conversation_outbox ALTER COLUMN thread_title DROP NOT NULL;
ALTER TABLE conversation_outbox ALTER COLUMN recipients DROP NOT NULL;

DROP TABLE conversation_projection;

-- A conversation's single thread - resource_url/thread_title/recipients
-- never change after ConversationStarted, so they stay columns here.
-- thread_author is the thread's frozen creator, split out so a reply's
-- authorship (rule 3 of "reply to a thread") can be checked without
-- assuming the opening message is always Messages[0].
CREATE TABLE conversation_projection (
    id            TEXT PRIMARY KEY,
    resource_url  TEXT NOT NULL,
    thread_id     TEXT NOT NULL,
    thread_title  TEXT NOT NULL,
    thread_author TEXT NOT NULL,
    recipients    TEXT[] NOT NULL
);

-- A thread's messages, one row per message, in append order via the
-- global event sequence each was projected from - see
-- docs/write-path.md and rule 7 of "reply to a thread".
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

ALTER TABLE conversation_outbox ALTER COLUMN recipients SET NOT NULL;
ALTER TABLE conversation_outbox ALTER COLUMN thread_title SET NOT NULL;
ALTER TABLE conversation_outbox ALTER COLUMN resource_url SET NOT NULL;
ALTER TABLE conversation_outbox ALTER COLUMN creator SET NOT NULL;
ALTER TABLE conversation_outbox DROP COLUMN event_type;

ALTER TABLE conversation_events ALTER COLUMN recipients SET NOT NULL;
ALTER TABLE conversation_events ALTER COLUMN thread_title SET NOT NULL;
ALTER TABLE conversation_events ALTER COLUMN resource_url SET NOT NULL;
ALTER TABLE conversation_events ALTER COLUMN creator SET NOT NULL;
ALTER TABLE conversation_events DROP COLUMN event_type;
