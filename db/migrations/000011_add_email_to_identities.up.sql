BEGIN;

ALTER TABLE identities
ADD COLUMN email text;

COMMIT;