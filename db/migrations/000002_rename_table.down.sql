BEGIN;

ALTER TABLE connections RENAME TO linked_accounts;

COMMIT;