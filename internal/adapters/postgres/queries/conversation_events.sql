-- name: InsertConversationCreatedEvent :one
INSERT INTO conversation_events (
    event_type, conversation_id, creator, resource_url, occurred_at
) VALUES (
    'ConversationCreated', $1, $2, $3, $4
)
RETURNING sequence;

-- name: InsertThreadStartedEvent :one
INSERT INTO conversation_events (
    event_type, conversation_id, thread_id, thread_title, author, recipients, occurred_at
) VALUES (
    'ThreadStarted', $1, $2, $3, $4, $5, $6
)
RETURNING sequence;

-- name: InsertMessagePostedEvent :one
INSERT INTO conversation_events (
    event_type, conversation_id, thread_id, message_id, author, message_text, occurred_at
) VALUES (
    'MessagePosted', $1, $2, $3, $4, $5, $6
)
RETURNING sequence;
