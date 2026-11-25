ALTER TABLE connections
ADD COLUMN last_synced_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN last_cursor TEXT;

CREATE INDEX IF NOT EXISTS idx_connections_last_synced ON connections (last_synced_at ASC NULLS FIRST);
