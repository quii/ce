-- +goose Up
-- Replaces thread_author + recipients with a single participants column -
-- the union of a thread's original author and its recipients, computed
-- once and frozen (rules 1-2 of "thread participants") - so the read model
-- has one column to project into and read back, matching
-- domain.ThreadView.Participants.
ALTER TABLE conversation_projection ADD COLUMN participants TEXT[];
UPDATE conversation_projection SET participants = array_prepend(thread_author, recipients);
ALTER TABLE conversation_projection ALTER COLUMN participants SET NOT NULL;
ALTER TABLE conversation_projection DROP COLUMN thread_author;
ALTER TABLE conversation_projection DROP COLUMN recipients;

-- +goose Down
ALTER TABLE conversation_projection ADD COLUMN thread_author TEXT;
ALTER TABLE conversation_projection ADD COLUMN recipients TEXT[];
UPDATE conversation_projection SET thread_author = participants[1], recipients = participants[2:array_length(participants, 1)];
ALTER TABLE conversation_projection ALTER COLUMN thread_author SET NOT NULL;
ALTER TABLE conversation_projection ALTER COLUMN recipients SET NOT NULL;
ALTER TABLE conversation_projection DROP COLUMN participants;
