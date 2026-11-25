BEGIN;

CREATE EXTENSION IF NOT EXISTS ulid;

-- Drop foreign keys so their UUID columns can be converted.
ALTER TABLE connections DROP CONSTRAINT linked_accounts_user_id_fkey;
ALTER TABLE activities DROP CONSTRAINT activities_user_id_fkey;
ALTER TABLE identities DROP CONSTRAINT identities_user_id_fkey;

-- Remove UUID-generating defaults from primary IDs.
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE connections ALTER COLUMN id DROP DEFAULT;
ALTER TABLE activities ALTER COLUMN id DROP DEFAULT;
ALTER TABLE identities ALTER COLUMN id DROP DEFAULT;

-- Convert IDs and the user_id foreign-key columns.
ALTER TABLE users
ALTER COLUMN id TYPE ulid USING id::ulid;

ALTER TABLE connections
ALTER COLUMN id TYPE ulid USING id::ulid,
ALTER COLUMN user_id TYPE ulid USING user_id::ulid;

ALTER TABLE activities
ALTER COLUMN id TYPE ulid USING id::ulid,
ALTER COLUMN user_id TYPE ulid USING user_id::ulid;

ALTER TABLE identities
ALTER COLUMN id TYPE ulid USING id::ulid,
ALTER COLUMN user_id TYPE ulid USING user_id::ulid;

-- Generate ULIDs for future rows.
ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_ulid();
ALTER TABLE connections ALTER COLUMN id SET DEFAULT gen_ulid();
ALTER TABLE activities ALTER COLUMN id SET DEFAULT gen_ulid();
ALTER TABLE identities ALTER COLUMN id SET DEFAULT gen_ulid();

-- Restore the foreign keys.
ALTER TABLE connections
ADD CONSTRAINT linked_accounts_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE activities
ADD CONSTRAINT activities_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE identities
ADD CONSTRAINT identities_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

COMMIT;
