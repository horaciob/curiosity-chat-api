CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('text', 'poi_share')),
    content TEXT,
    poi_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT messages_content_check CHECK (
        (type = 'text' AND content IS NOT NULL AND poi_id IS NULL) OR
        (type = 'poi_share' AND poi_id IS NOT NULL AND content IS NULL)
    )
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC);
