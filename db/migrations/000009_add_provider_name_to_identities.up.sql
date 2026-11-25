BEGIN;

ALTER TABLE identities
ADD COLUMN provider text NOT NULL;

COMMIT;