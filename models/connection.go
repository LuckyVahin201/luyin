package models

import (
	"context"
	"time"

	"luyin/db"

	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"
)

type Connection struct {
	ID           ulid.ULID     `json:"id"`
	UserID       ulid.ULID     `json:"-"`
	Provider     string     `json:"provider"`
	AccessToken  string     `json:"-"`
	RefreshToken *string    `json:"-"`
	ExpiresAt    *int64     `json:"expires_at"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	LastCursor   *string    `json:"-"`
	Invalidated  bool       `json:"invalidated"`
}

func (m *Connection) FromRow(row pgx.Row) error {
	return row.Scan(
		&m.ID,
		&m.UserID,
		&m.Provider,
		&m.AccessToken,
		&m.RefreshToken,
		&m.ExpiresAt,
		&m.LastSyncedAt,
		&m.LastCursor,
		&m.Invalidated,
	)
}

func (c *Connection) NeedsRefresh(buffer time.Duration) bool {
	if c.ExpiresAt == nil || *c.ExpiresAt == 0 {
		return false
	}
	return time.Now().Add(buffer).Unix() >= *c.ExpiresAt
}

func (u *User) UpsertConnection(ctx context.Context, conn *Connection) (*Connection, error) {
	var saved Connection
	err := saved.FromRow(db.Pool.QueryRow(ctx, `
		INSERT INTO connections (user_id, provider, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5)
		ON CONFLICT (user_id, provider) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expires_at = EXCLUDED.expires_at,
			invalidated = FALSE
		RETURNING *
	`, u.ID, conn.Provider, conn.AccessToken, conn.RefreshToken, conn.ExpiresAt))

	if err != nil {
		return nil, err
	}

	return &saved, nil
}

func (c *Connection) UpdateTokens(ctx context.Context, accessToken string, refreshToken *string, expiresAt *int64) error {
	return c.FromRow(db.Pool.QueryRow(ctx, `
		UPDATE connections
		SET access_token = $1,
		    refresh_token = COALESCE($2, refresh_token),
		    expires_at = $3,
		    invalidated = FALSE
		WHERE id = $4
		RETURNING *
	`, accessToken, refreshToken, expiresAt, c.ID))
}

func (u *User) GetConnections(ctx context.Context) ([]Connection, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT * FROM connections WHERE user_id = $1
	`, u.ID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var conns []Connection
	for rows.Next() {
		var c Connection
		if err := c.FromRow(rows); err != nil {
			return nil, err
		}

		conns = append(conns, c)
	}

	return conns, rows.Err()
}

func (u *User) HasStaleConnections(ctx context.Context, staleBefore time.Time) (bool, error) {
	var exists bool

	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM connections
			WHERE user_id = $1
			  AND invalidated = FALSE
			  AND (last_synced_at IS NULL OR last_synced_at < $2)
		)
	`, u.ID, staleBefore).Scan(&exists)

	return exists, err
}

func (c *Connection) MarkSynced(ctx context.Context, nextCursor string) error {
	return c.FromRow(db.Pool.QueryRow(ctx, `
		UPDATE connections
		SET last_synced_at = NOW(),
		    last_cursor = COALESCE(NULLIF($1, ''), last_cursor)
		WHERE id = $2
		RETURNING *
	`, nextCursor, c.ID))
}

func (c *Connection) MarkInvalidated(ctx context.Context) error {
	return c.FromRow(db.Pool.QueryRow(ctx, `
		UPDATE connections
		SET invalidated = TRUE
		WHERE id = $1
		RETURNING *
	`, c.ID))
}

func GetStaleConnectionUserIDs(ctx context.Context, staleBefore time.Time, limit int) ([]string, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT user_id
		FROM connections
		WHERE invalidated = FALSE
		  AND (last_synced_at IS NULL OR last_synced_at < $1)
		LIMIT $2
	`, staleBefore, limit)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}

		userIDs = append(userIDs, uid)
	}

	return userIDs, rows.Err()
}
