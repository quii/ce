-- name: InsertConversationEvent :one
INSERT INTO conversation_events (
    conversation_id, thread_id, message_id, creator, resource_url, thread_title, author, recipients, message_text, occurred_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING sequence;
