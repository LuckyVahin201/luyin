BEGIN;

ALTER TABLE linked_accounts RENAME TO connections;

COMMIT;