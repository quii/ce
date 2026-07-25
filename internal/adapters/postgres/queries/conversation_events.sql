-- name: InsertEvent :one
INSERT INTO conversation_events (
    event_type, conversation_id, occurred_at, payload
) VALUES (
    $1, $2, $3, $4
)
RETURNING sequence;

-- name: ListConversationEvents :many
SELECT sequence, event_type, conversation_id, occurred_at, payload
FROM conversation_events
WHERE conversation_id = $1
ORDER BY sequence;
