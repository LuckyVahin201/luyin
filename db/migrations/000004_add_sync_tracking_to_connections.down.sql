ALTER TABLE connections
DROP COLUMN IF EXISTS last_synced_at,
DROP COLUMN IF EXISTS last_cursor;
