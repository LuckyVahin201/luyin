ALTER TABLE connections
ADD COLUMN invalidated BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_connections_active_stale
ON connections (last_synced_at ASC NULLS FIRST)
WHERE invalidated = FALSE;
