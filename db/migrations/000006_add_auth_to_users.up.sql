BEGIN;

ALTER TABLE users
ADD COLUMN password_hash TEXT;

-- this is for oidc connections in the future like google and whatnot
CREATE TABLE IF NOT EXISTS identities (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issuer text NOT NULL,
    subject text NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (issuer, subject)
);

CREATE INDEX identities_user_id_idx ON identities(user_id);

COMMIT;