package models

import (
	"context"
	"time"

	"luyin/db"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (m *User) FromRow(row pgx.Row) error {
	return row.Scan(
		&m.ID,
		&m.Email,
		&m.CreatedAt,
	)
}

func GetUser(ctx context.Context, id string) (*User, error) {
	var u User

	err := u.FromRow(db.Pool.QueryRow(ctx, `
		SELECT id, email, created_at
		FROM users
		WHERE id = $1
	`, id))

	if err != nil {
		return nil, err
	}

	return &u, nil
}
