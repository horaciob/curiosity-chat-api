DROP INDEX IF EXISTS idx_messages_conversation_unread;

DROP INDEX idx_conversations_user1;
CREATE INDEX idx_conversations_user1 ON conversations(user1_id, last_message_at DESC);

DROP INDEX idx_conversations_user2;
CREATE INDEX idx_conversations_user2 ON conversations(user2_id, last_message_at DESC);
