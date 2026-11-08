package db

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func newProvider(db *sql.DB) (*goose.Provider, error) {
	migrations, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectSQLite3, db, migrations)
}

// Migrate runs all pending migrations on the given database.
func Migrate(db *sql.DB) error {
	provider, err := newProvider(db)
	if err != nil {
		return err
	}
	_, err = provider.Up(context.Background())
	return err
}

// NeedsUpgrade checks if there are any pending migrations for the database.
func NeedsUpgrade(db *sql.DB) (bool, error) {
	provider, err := newProvider(db)
	if err != nil {
		return false, err
	}
	return provider.HasPending(context.Background())
}
