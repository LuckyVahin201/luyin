DROP INDEX IF EXISTS idx_connections_active_stale;

ALTER TABLE connections
DROP COLUMN IF EXISTS invalidated;
