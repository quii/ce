-- +goose Up
-- A conversation is no longer assumed to have exactly one thread (rule 9
-- of "add a thread to a conversation") - conversation_projection's
-- thread_id/thread_title/participants columns move out into their own
-- table, one row per thread, ordered by the global event sequence its
-- ThreadStarted was applied at (rule 10: creation order).
--
-- This is a destructive rewrite of a table that already carries one row
-- per existing conversation, with no backfill from the dropped columns
-- into the new table below and a NOT NULL conversation_projection_messages
-- column added with no default. Accepted per the same "nothing has shipped
-- yet" pre-release exception docs/adr/0026-sql-spec-first-with-sqlc.md's
-- Consequences section documents for migration 00001: there is no real
-- data anywhere, only test fixtures recreated from scratch by every test
-- run, so there's nothing to carry forward. This exception is retired the
-- moment there's a real deployment with real data to lose.
ALTER TABLE conversation_projection
    DROP COLUMN thread_id,
    DROP COLUMN thread_title,
    DROP COLUMN participants;

-- A thread's id is scoped to its conversation, not globally unique on its
-- own - PRIMARY KEY (conversation_id, id) rather than a plain PRIMARY KEY
-- (id), so an id collision between two unrelated conversations can never
-- silently overwrite (or no-op against) another conversation's thread the
-- way a global-id-only key would. This restores the same structural
-- guarantee the pre-story schema had for free (one thread stored per
-- conversation row, keyed by the conversation's own id) now that a
-- conversation can have more than one thread.
CREATE TABLE conversation_projection_threads (
    id              TEXT NOT NULL,
    conversation_id TEXT NOT NULL REFERENCES conversation_projection (id),
    sequence        BIGINT NOT NULL,
    title           TEXT NOT NULL,
    participants    TEXT[] NOT NULL,
    PRIMARY KEY (conversation_id, id)
);

CREATE INDEX conversation_projection_threads_conversation_id_idx
    ON conversation_projection_threads (conversation_id, sequence);

-- A message now belongs to a specific thread, not just a conversation -
-- there can be more than one thread to attribute it to. The compound FK
-- (matching conversation_id together with thread_id) is what stops a
-- message from ever pointing at a thread belonging to a different
-- conversation than the message's own conversation_id.
ALTER TABLE conversation_projection_messages
    ADD COLUMN thread_id TEXT NOT NULL,
    ADD CONSTRAINT conversation_projection_messages_thread_id_fkey
        FOREIGN KEY (conversation_id, thread_id)
        REFERENCES conversation_projection_threads (conversation_id, id);

-- +goose Down
ALTER TABLE conversation_projection_messages
    DROP CONSTRAINT conversation_projection_messages_thread_id_fkey,
    DROP COLUMN thread_id;

DROP TABLE conversation_projection_threads;

ALTER TABLE conversation_projection
    ADD COLUMN thread_id    TEXT,
    ADD COLUMN thread_title TEXT,
    ADD COLUMN participants TEXT[];
