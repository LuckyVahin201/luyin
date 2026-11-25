package models

import (
	"context"
	"time"

	"luyin/db"

	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"
)

type Activity struct {
	ID             ulid.ULID `json:"id"`
	UserID         ulid.ULID `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderItemID string    `json:"provider_item_id"`
	Action         string    `json:"action"`
	Title          string    `json:"title"`
	OccurredAt     time.Time `json:"occurred_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func (m *Activity) FromRow(row pgx.Row) error {
	return row.Scan(
		&m.ID,
		&m.UserID,
		&m.Provider,
		&m.ProviderItemID,
		&m.Action,
		&m.Title,
		&m.OccurredAt,
		&m.CreatedAt,
	)
}

func (u *User) CreateActivity(ctx context.Context, act *Activity) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO activities (user_id, provider, provider_item_id, action, title, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, provider, provider_item_id, occurred_at) DO NOTHING
	`, u.ID, act.Provider, act.ProviderItemID, act.Action, act.Title, act.OccurredAt)

	return err
}

func (u *User) GetActivities(ctx context.Context, limit int) ([]Activity, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, provider, provider_item_id, action, title, occurred_at, created_at
		FROM activities
		WHERE user_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2
	`, u.ID, limit)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		if err := a.FromRow(rows); err != nil {
			return nil, err
		}

		activities = append(activities, a)
	}

	return activities, rows.Err()
}
