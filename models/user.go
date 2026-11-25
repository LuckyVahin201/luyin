package models

import (
	"context"
	"time"

	"luyin/db"
	"luyin/utils"

	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"
)

type User struct {
	ID        ulid.ULID    `json:"id"`
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

func CreateUser(ctx context.Context, email string, password_hash string) (*User, error) {
	user := User{} 

	err := user.FromRow(db.Pool.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash)
		
		RETURNING *
	`,utils.ToUUID(ulid.Make()), email, password_hash))

	return &user, err
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