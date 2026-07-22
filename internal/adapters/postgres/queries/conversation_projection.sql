-- name: ApplyConversationStartedProjection :exec
INSERT INTO conversation_projection (
    id, resource_url, thread_id, thread_title, participants
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (id) DO UPDATE SET
    resource_url = EXCLUDED.resource_url,
    thread_id = EXCLUDED.thread_id,
    thread_title = EXCLUDED.thread_title,
    participants = EXCLUDED.participants;

-- name: AppendConversationProjectionMessage :exec
INSERT INTO conversation_projection_messages (
    conversation_id, sequence, author, message_text, posted_at
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (conversation_id, sequence) DO NOTHING;

-- name: GetConversationProjection :one
SELECT id, resource_url, thread_id, thread_title, participants
FROM conversation_projection
WHERE id = $1;

-- name: ListConversationProjectionMessages :many
SELECT conversation_id, sequence, author, message_text, posted_at
FROM conversation_projection_messages
WHERE conversation_id = $1
ORDER BY sequence;

-- name: SetProjectionCheckpoint :exec
UPDATE projection_checkpoint SET sequence = $1 WHERE sequence < $1;

-- name: GetProjectionCheckpoint :one
SELECT sequence FROM projection_checkpoint LIMIT 1;
