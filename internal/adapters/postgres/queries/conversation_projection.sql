-- name: ApplyConversationCreatedProjection :exec
INSERT INTO conversation_projection (
    id, resource_url
) VALUES (
    $1, $2
)
ON CONFLICT (id) DO UPDATE SET
    resource_url = EXCLUDED.resource_url;

-- name: ApplyThreadStartedProjection :exec
-- No ON CONFLICT here, deliberately: (conversation_id, id) is the table's
-- primary key, so a genuine collision - the same thread id landing twice
-- against the same conversation with different data - fails loudly as a
-- unique-violation error instead of silently no-opping.
INSERT INTO conversation_projection_threads (
    id, conversation_id, sequence, title, participants
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: AppendConversationProjectionMessage :exec
INSERT INTO conversation_projection_messages (
    conversation_id, thread_id, sequence, author, message_text, posted_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (conversation_id, sequence) DO NOTHING;

-- name: GetConversationProjection :one
SELECT id, resource_url
FROM conversation_projection
WHERE id = $1;

-- name: ConversationProjectionExists :one
SELECT EXISTS (
    SELECT 1 FROM conversation_projection_threads WHERE conversation_id = $1
);

-- name: ListConversationProjectionThreads :many
SELECT id, title, participants
FROM conversation_projection_threads
WHERE conversation_id = $1
ORDER BY sequence;

-- name: ListConversationProjectionMessages :many
SELECT thread_id, sequence, author, message_text, posted_at
FROM conversation_projection_messages
WHERE conversation_id = $1
ORDER BY sequence;

-- name: SetProjectionCheckpoint :exec
UPDATE projection_checkpoint SET sequence = $1 WHERE sequence < $1;

-- name: GetProjectionCheckpoint :one
SELECT sequence FROM projection_checkpoint LIMIT 1;

-- name: ListConversationIDsByParticipant :many
-- Returns distinct conversation IDs where the participant appears in at
-- least one thread, ordered by the latest message posted to any of the
-- participant's visible threads within that conversation (most-recently-
-- active first) - rule 3 of "get conversations by participant".
SELECT cp.id AS conversation_id,
       COALESCE(
           MAX(cpm.posted_at) FILTER (WHERE cpt2.participants @> ARRAY[$1::text]),
           '-infinity'::timestamptz
       ) AS latest_at
FROM conversation_projection cp
JOIN conversation_projection_threads cpt ON cpt.conversation_id = cp.id
     AND cpt.participants @> ARRAY[$1::text]
LEFT JOIN conversation_projection_threads cpt2 ON cpt2.conversation_id = cp.id
     AND cpt2.participants @> ARRAY[$1::text]
LEFT JOIN conversation_projection_messages cpm ON cpm.conversation_id = cpt2.conversation_id
     AND cpm.thread_id = cpt2.id
GROUP BY cp.id
ORDER BY latest_at DESC;

-- name: ListParticipantThreadsForConversation :many
-- Returns all threads in a conversation that the participant is part of,
-- in creation order (rule 2 of "get conversations by participant").
SELECT id, title, participants
FROM conversation_projection_threads
WHERE conversation_id = $1
  AND participants @> ARRAY[$2::text]
ORDER BY sequence;

-- name: ListParticipantMessagesForConversation :many
-- Returns all messages for threads in a conversation that the participant
-- is part of, in posting order.
SELECT cpm.thread_id, cpm.sequence, cpm.author, cpm.message_text, cpm.posted_at
FROM conversation_projection_messages cpm
WHERE cpm.conversation_id = $1
  AND cpm.thread_id IN (
      SELECT cpt.id FROM conversation_projection_threads cpt
      WHERE cpt.conversation_id = $1
        AND cpt.participants @> ARRAY[$2::text]
  )
ORDER BY cpm.sequence;
