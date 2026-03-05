ALTER TABLE messages
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'sent'
        CHECK (status IN ('sent', 'delivered', 'read'));
