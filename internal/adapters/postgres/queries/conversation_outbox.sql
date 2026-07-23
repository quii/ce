-- name: EnqueueOutboxEntry :exec
INSERT INTO conversation_outbox (
    sequence, event_type, conversation_id, occurred_at, payload
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (sequence) DO NOTHING;

-- name: ListPendingOutboxEntries :many
SELECT sequence, event_type, conversation_id, occurred_at, payload
FROM conversation_outbox
WHERE done_at IS NULL
ORDER BY sequence;

-- name: MarkOutboxEntryDone :exec
UPDATE conversation_outbox SET done_at = now() WHERE sequence = $1;
