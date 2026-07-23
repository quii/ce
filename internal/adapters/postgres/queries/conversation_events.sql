-- name: InsertEvent :one
INSERT INTO conversation_events (
    event_type, conversation_id, occurred_at, payload
) VALUES (
    $1, $2, $3, $4
)
RETURNING sequence;
