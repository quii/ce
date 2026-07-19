-- name: EnqueueOutboxEntry :exec
INSERT INTO conversation_outbox (
    sequence, conversation_id, thread_id, message_id, creator, resource_url, thread_title, author, recipients, message_text, occurred_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (sequence) DO NOTHING;

-- name: ListPendingOutboxEntries :many
SELECT sequence, conversation_id, thread_id, message_id, creator, resource_url, thread_title, author, recipients, message_text, occurred_at
FROM conversation_outbox
WHERE done_at IS NULL
ORDER BY sequence;

-- name: MarkOutboxEntryDone :exec
UPDATE conversation_outbox SET done_at = now() WHERE sequence = $1;
