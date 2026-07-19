-- name: ApplyConversationProjection :exec
INSERT INTO conversation_projection (
    id, resource_url, thread_id, thread_title, recipients, message_author, message_text, message_posted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (id) DO UPDATE SET
    resource_url = EXCLUDED.resource_url,
    thread_id = EXCLUDED.thread_id,
    thread_title = EXCLUDED.thread_title,
    recipients = EXCLUDED.recipients,
    message_author = EXCLUDED.message_author,
    message_text = EXCLUDED.message_text,
    message_posted_at = EXCLUDED.message_posted_at;

-- name: GetConversationProjection :one
SELECT id, resource_url, thread_id, thread_title, recipients, message_author, message_text, message_posted_at
FROM conversation_projection
WHERE id = $1;

-- name: SetProjectionCheckpoint :exec
UPDATE projection_checkpoint SET sequence = $1 WHERE sequence < $1;

-- name: GetProjectionCheckpoint :one
SELECT sequence FROM projection_checkpoint LIMIT 1;
