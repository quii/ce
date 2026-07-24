-- name: ApplyConversationCreatedProjection :exec
INSERT INTO conversation_projection (
    id, resource_url
) VALUES (
    $1, $2
)
ON CONFLICT (id) DO UPDATE SET
    resource_url = EXCLUDED.resource_url;

-- name: ApplyThreadStartedProjection :exec
-- No ON CONFLICT here, deliberately: (conversation_id, id) is the table's
-- primary key, so a genuine collision - the same thread id landing twice
-- against the same conversation with different data - fails loudly as a
-- unique-violation error instead of silently no-opping.
INSERT INTO conversation_projection_threads (
    id, conversation_id, sequence, title, participants
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: AppendConversationProjectionMessage :exec
INSERT INTO conversation_projection_messages (
    conversation_id, thread_id, sequence, author, message_text, posted_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (conversation_id, sequence) DO NOTHING;

-- name: GetConversationProjection :one
SELECT id, resource_url
FROM conversation_projection
WHERE id = $1;

-- name: ConversationProjectionExists :one
SELECT EXISTS (
    SELECT 1 FROM conversation_projection_threads WHERE conversation_id = $1
);

-- name: ListConversationProjectionThreads :many
SELECT id, title, participants
FROM conversation_projection_threads
WHERE conversation_id = $1
ORDER BY sequence;

-- name: ListConversationProjectionMessages :many
SELECT thread_id, sequence, author, message_text, posted_at
FROM conversation_projection_messages
WHERE conversation_id = $1
ORDER BY sequence;

-- name: SetProjectionCheckpoint :exec
UPDATE projection_checkpoint SET sequence = $1 WHERE sequence < $1;

-- name: GetProjectionCheckpoint :one
SELECT sequence FROM projection_checkpoint LIMIT 1;
