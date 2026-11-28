BEGIN;

CREATE TABLE IF NOT EXISTS pending_identities (
    token text PRIMARY KEY,
    preferred_username VARCHAR(50),
    provider text NOT NULL,
    issuer text NOT NULL,
    subject text NOT NULL,
    email text NOT NULL,
    expires_at timestamptz NOT NULL,

    UNIQUE (issuer, subject)
);

COMMIT;