BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

ALTER TABLE connections DROP CONSTRAINT linked_accounts_user_id_fkey;
ALTER TABLE activities DROP CONSTRAINT activities_user_id_fkey;
ALTER TABLE identities DROP CONSTRAINT identities_user_id_fkey;

ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE connections ALTER COLUMN id DROP DEFAULT;
ALTER TABLE activities ALTER COLUMN id DROP DEFAULT;
ALTER TABLE identities ALTER COLUMN id DROP DEFAULT;

ALTER TABLE users ALTER COLUMN id TYPE uuid USING id::uuid;

ALTER TABLE connections
ALTER COLUMN id TYPE uuid USING id::uuid,
ALTER COLUMN user_id TYPE uuid USING user_id::uuid;

ALTER TABLE activities
ALTER COLUMN id TYPE uuid USING id::uuid,
ALTER COLUMN user_id TYPE uuid USING user_id::uuid;

ALTER TABLE identities
ALTER COLUMN id TYPE uuid USING id::uuid,
ALTER COLUMN user_id TYPE uuid USING user_id::uuid;

ALTER TABLE users ALTER COLUMN id SET DEFAULT uuid_generate_v4();
ALTER TABLE connections ALTER COLUMN id SET DEFAULT uuid_generate_v4();
ALTER TABLE activities ALTER COLUMN id SET DEFAULT uuid_generate_v4();
ALTER TABLE identities ALTER COLUMN id SET DEFAULT uuid_generate_v4();

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
-- holy moly
-- this is why i wanted to just change old migrations :/

-- well if you want you can squash it down now since theres not much data loss 
-- idk how to squash it down :sob:

-- should i
-- nah we can put it off fr tech debt:tm:-- holy moly
-- this is why i wanted to just change old migrations :/

-- well if you want you can squash it down now since theres not much data loss 
-- idk how to squash it down :sob:

-- should i
-- nah we can put it off fr tech debt:tm: