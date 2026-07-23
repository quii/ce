-- name: ApplyConversationCreatedProjection :exec
INSERT INTO conversation_projection (
    id, resource_url
) VALUES (
    $1, $2
)
ON CONFLICT (id) DO UPDATE SET
    resource_url = EXCLUDED.resource_url;

-- name: ApplyThreadStartedProjection :exec
INSERT INTO thread_projection (
    id, conversation_id, title, participants
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (id) DO UPDATE SET
    conversation_id = EXCLUDED.conversation_id,
    title = EXCLUDED.title,
    participants = EXCLUDED.participants;

-- name: AppendConversationProjectionMessage :exec
INSERT INTO conversation_projection_messages (
    conversation_id, sequence, author, message_text, posted_at
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (conversation_id, sequence) DO NOTHING;

-- name: GetConversationProjection :one
SELECT c.id, c.resource_url, t.id AS thread_id, t.title AS thread_title, t.participants
FROM conversation_projection c
JOIN thread_projection t ON t.conversation_id = c.id
WHERE c.id = $1;

-- name: ListConversationProjectionMessages :many
SELECT conversation_id, sequence, author, message_text, posted_at
FROM conversation_projection_messages
WHERE conversation_id = $1
ORDER BY sequence;

-- name: SetProjectionCheckpoint :exec
UPDATE projection_checkpoint SET sequence = $1 WHERE sequence < $1;

-- name: GetProjectionCheckpoint :one
SELECT sequence FROM projection_checkpoint LIMIT 1;
