package db

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// Migrate runs all pending migrations on the given database.
func Migrate(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

// NeedsUpgrade checks if there are any pending migrations for the database.
func NeedsUpgrade(db *sql.DB) (bool, error) {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return false, err
	}

	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		// If the migration version table doesn't exist yet, it definitely needs an upgrade
		return true, nil
	}

	migrations, err := goose.CollectMigrations("migrations", currentVersion, (1<<63)-1)
	if err != nil {
		if errors.Is(err, goose.ErrNoNextVersion) || errors.Is(err, goose.ErrNoMigrationFiles) {
			return false, nil
		}
		return false, err
	}
	return len(migrations) > 0, nil
}
