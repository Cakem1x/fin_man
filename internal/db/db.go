package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Cakem1x/fin_man/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Open(dbPath string) (*DB, error) {
	// Enable Write-Ahead Logging for better concurrency/resilience
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

type DuplicatePair struct {
	Existing model.Transaction
	New      model.Transaction
}

// InsertTransactions inserts a batch of transactions into the database.
// It uses ON CONFLICT(id) DO NOTHING to ignore duplicates.
func (db *DB) InsertTransactions(ctx context.Context, txs []model.Transaction) (int, []DuplicatePair, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO transactions (id, date, payee, amount_cents, currency, memo, archive_file_path)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING;
	`)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	selectStmt, err := tx.PrepareContext(ctx, `
		SELECT id, date, payee, amount_cents, currency, memo, archive_file_path
		FROM transactions
		WHERE id = ?
	`)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer func() { _ = selectStmt.Close() }()

	insertedCount := 0
	var duplicates []DuplicatePair

	for _, t := range txs {
		res, err := stmt.ExecContext(ctx, t.ID, t.Date, t.Payee, t.AmountCents, t.Currency, t.Memo, t.ArchiveFilePath)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to insert transaction %s: %w", t.ID, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return 0, nil, fmt.Errorf("failed to get rows affected: %w", err)
		}

		if affected == 1 {
			insertedCount++
		} else {
			row := selectStmt.QueryRowContext(ctx, t.ID)
			var existing model.Transaction
			err := row.Scan(&existing.ID, &existing.Date, &existing.Payee, &existing.AmountCents, &existing.Currency, &existing.Memo, &existing.ArchiveFilePath)
			if err != nil {
				return 0, nil, fmt.Errorf("failed to fetch existing transaction %s: %w", t.ID, err)
			}
			duplicates = append(duplicates, DuplicatePair{
				Existing: existing,
				New:      t,
			})
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	if len(duplicates) > 0 {
		return insertedCount, duplicates, fmt.Errorf("found %d duplicate transactions", len(duplicates))
	}

	return insertedCount, nil, nil
}
