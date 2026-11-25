BEGIN;

DROP INDEX IF EXISTS identities_user_id_idx;
DROP TABLE IF EXISTS identities;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;

COMMIT;