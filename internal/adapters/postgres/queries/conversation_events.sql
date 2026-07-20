-- name: InsertConversationStartedEvent :one
INSERT INTO conversation_events (
    event_type, conversation_id, thread_id, message_id, creator, resource_url, thread_title, author, recipients, message_text, occurred_at
) VALUES (
    'ConversationStarted', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING sequence;

-- name: InsertReplyPostedEvent :one
INSERT INTO conversation_events (
    event_type, conversation_id, thread_id, message_id, author, message_text, occurred_at
) VALUES (
    'ReplyPosted', $1, $2, $3, $4, $5, $6
)
RETURNING sequence;
