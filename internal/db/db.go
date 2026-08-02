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

// GetUncategorizedTransactions fetches transactions that have no category assigned.
func (db *DB) GetUncategorizedTransactions(ctx context.Context) ([]model.Transaction, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, date, payee, amount_cents, currency, memo, archive_file_path, category_id
		FROM transactions
		WHERE category_id IS NULL
		ORDER BY date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var txs []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.Date, &t.Payee, &t.AmountCents, &t.Currency, &t.Memo, &t.ArchiveFilePath, &t.CategoryID); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

func (db *DB) GetAllCategories(ctx context.Context) ([]model.Category, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM categories ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var cats []model.Category
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (db *DB) GetAllTags(ctx context.Context) ([]model.Tag, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// GetAllTransactions fetches all transactions, both categorized and uncategorized.
func (db *DB) GetAllTransactions(ctx context.Context) ([]model.Transaction, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.date, t.payee, t.amount_cents, t.currency, t.memo, t.archive_file_path, t.category_id, c.name
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		ORDER BY t.date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var txs []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.Date, &t.Payee, &t.AmountCents, &t.Currency, &t.Memo, &t.ArchiveFilePath, &t.CategoryID, &t.CategoryName); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Note: We don't hydrate tags here yet for simplicity in this draft,
		// but we would typically run a second query or JOIN to fetch tags.

		txs = append(txs, t)
	}
	return txs, rows.Err()
}
