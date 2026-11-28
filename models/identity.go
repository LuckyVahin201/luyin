package models

import (
	"context"
	"luyin/db"
	"luyin/utils"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"
)

type Identity struct {
	ID        ulid.ULID `json:"id"`
	UserID    ulid.ULID `json:"user_id"`
	Provider  string    `json:"provider"`
	Issuer    string    `json:"issuer"`
	Subject   string    `json:"subject"`
	CreatedAt time.Time `json:"created_at"`
}

type PendingIdentity struct {
	Token             string    `json:"token"`
	PreferredUsername *string   `json:"preferred_username"`
	Provider          string    `json:"provider"`
	Issuer            string    `json:"-"`
	Subject           string    `json:"-"`
	Email             string    `json:"email"`
	ExpiresAt         time.Time `json:"expires_at"`
}

func (m *Identity) FromRow(row pgx.Row) error {
	return row.Scan(
		&m.ID,
		&m.UserID,
		&m.Provider,
		&m.Issuer,
		&m.Subject,
		&m.CreatedAt,
	)
}

func (m *PendingIdentity) FromRow(row pgx.Row) error {
	return row.Scan(
		&m.Token,
		&m.PreferredUsername,
		&m.Provider,
		&m.Issuer,
		&m.Subject,
		&m.Email,
		&m.ExpiresAt,
	)
}

func UpsertPendingIdentity(ctx context.Context, preferredUsername *string, provider, issuer, subject, email string) (*PendingIdentity, error) {
	p := PendingIdentity{}

	err := p.FromRow(db.Pool.QueryRow(ctx, `
		INSERT INTO pending_identities (token, preferred_username, provider, issuer, subject, email, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, now() + interval '10 minutes')

		ON CONFLICT (issuer, subject) DO UPDATE
			SET token = EXCLUDED.token,
				preferred_username = EXCLUDED.preferred_username,
				email = EXCLUDED.email,
				expires_at = EXCLUDED.expires_at
		RETURNING token, preferred_username, provider, issuer, subject, email, expires_at
	`, utils.GenerateCodeVerifier(), preferredUsername, provider, issuer, subject, email))
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func GetPendingIdentityByToken(ctx context.Context, token string) (*PendingIdentity, error) {
	var p PendingIdentity

	err := p.FromRow(db.Pool.QueryRow(ctx, `
		SELECT token, preferred_username, provider, issuer, subject, email, expires_at
		FROM pending_identities
		WHERE token = $1 AND expires_at > now()
	`, token))
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func GetIdentityByIssuerAndSubject(ctx context.Context, issuer, subject string) (*Identity, error) {
	var i Identity

	err := i.FromRow(db.Pool.QueryRow(ctx, `
		SELECT id, user_id, provider, issuer, subject, created_at
		FROM identities
		WHERE issuer = $1 AND subject = $2
	`, issuer, subject))
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func UpdateIdentityEmail(ctx context.Context, id ulid.ULID, email string) (*Identity, error) {
    var i Identity

    err := i.FromRow(db.Pool.QueryRow(ctx, `
        UPDATE identities
        SET email = $2
        WHERE id = $1
        RETURNING id, user_id, provider, issuer, subject, email, created_at
    `, id.String(), email))
    if err != nil {
        return nil, err
    }

    return &i, nil
}
