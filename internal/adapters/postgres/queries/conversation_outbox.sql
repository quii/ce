-- name: EnqueueConversationCreatedOutboxEntry :exec
INSERT INTO conversation_outbox (
    sequence, event_type, conversation_id, creator, resource_url, occurred_at
) VALUES (
    $1, 'ConversationCreated', $2, $3, $4, $5
)
ON CONFLICT (sequence) DO NOTHING;

-- name: EnqueueThreadStartedOutboxEntry :exec
INSERT INTO conversation_outbox (
    sequence, event_type, conversation_id, thread_id, thread_title, author, recipients, occurred_at
) VALUES (
    $1, 'ThreadStarted', $2, $3, $4, $5, $6, $7
)
ON CONFLICT (sequence) DO NOTHING;

-- name: EnqueueMessagePostedOutboxEntry :exec
INSERT INTO conversation_outbox (
    sequence, event_type, conversation_id, thread_id, message_id, author, message_text, occurred_at
) VALUES (
    $1, 'MessagePosted', $2, $3, $4, $5, $6, $7
)
ON CONFLICT (sequence) DO NOTHING;

-- name: ListPendingOutboxEntries :many
SELECT sequence, event_type, conversation_id, thread_id, message_id, creator, resource_url, thread_title, author, recipients, message_text, occurred_at
FROM conversation_outbox
WHERE done_at IS NULL
ORDER BY sequence;

-- name: MarkOutboxEntryDone :exec
UPDATE conversation_outbox SET done_at = now() WHERE sequence = $1;
