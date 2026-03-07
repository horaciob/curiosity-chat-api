ALTER TABLE messages DROP CONSTRAINT messages_content_check;

ALTER TABLE messages ADD CONSTRAINT messages_content_check CHECK (
    (type = 'text'      AND content IS NOT NULL AND poi_id IS NULL)
    OR
    (type = 'poi_share' AND poi_id IS NOT NULL AND content IS NULL)
);
