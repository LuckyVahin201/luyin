package models

import (
	"context"
	"time"

	"luyin/db"

	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"
)

type User struct {
	ID           ulid.ULID `json:"id"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash *string   `json:"-"`
	Username     string    `json:"username"`
}

func (m *User) FromRow(row pgx.Row) error {
	return row.Scan(
		&m.ID,
		&m.Email,
		&m.CreatedAt,
		&m.PasswordHash,
		&m.Username,
	)
}

func CreateUserWithPassword(ctx context.Context, email, username, password_hash string) (*User, error) {
	user := User{}

	err := user.FromRow(db.Pool.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING *
	`, email, username, password_hash))

	return &user, err
}

func GetUser(ctx context.Context, id ulid.ULID) (*User, error) {
	var u User

	err := u.FromRow(db.Pool.QueryRow(ctx, `
		SELECT id, email, created_at, password_hash, username
		FROM users
		WHERE id = $1
	`, id.String()))

	if err != nil {
		return nil, err
	}

	return &u, nil
}

func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User

	err := u.FromRow(db.Pool.QueryRow(ctx, `
		SELECT id, email, created_at, password_hash, username
		FROM users
		WHERE email = $1
	`, email))

	if err != nil {
		return nil, err
	}

	return &u, nil
}
