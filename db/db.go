package db

import (
	ctx "context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect(url string) error {
	var err error

	Pool, err = pgxpool.New(ctx.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to connect to the database: %w", err)
	}

	return Pool.Ping(ctx.Background())
}

func Migrate(url string) error {
	m, err := migrate.New("file://db/migrations", url)

	if err != nil {
		return fmt.Errorf("migration init failed: %w", err)
	}
	defer m.Close()

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}
