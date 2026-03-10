-- Replace the simple last_message_at indices with functional composites using COALESCE.
-- ListByUser orders by COALESCE(last_message_at, created_at) DESC, which the old indices
-- could not serve — PostgreSQL had to re-sort after the bitmap scan.
-- With these functional composites, each index scan is already ordered, allowing a merge
-- instead of a full sort.
DROP INDEX idx_conversations_user1;
CREATE INDEX idx_conversations_user1
    ON conversations (user1_id, COALESCE(last_message_at, created_at) DESC);

DROP INDEX idx_conversations_user2;
CREATE INDEX idx_conversations_user2
    ON conversations (user2_id, COALESCE(last_message_at, created_at) DESC);

-- Partial index for MarkConversationRead.
-- The UPDATE scans: WHERE conversation_id = $1 AND sender_id != $2 AND status != 'read'.
-- This partial index narrows the scan to only unread messages per conversation,
-- avoiding a full scan of all messages in a conversation.
CREATE INDEX idx_messages_conversation_unread
    ON messages (conversation_id, sender_id)
    WHERE status != 'read';
