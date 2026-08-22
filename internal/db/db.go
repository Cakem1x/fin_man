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
		SELECT id, date, payee, amount_cents, currency, memo, archive_file_path, is_reviewed
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
			err := row.Scan(&existing.ID, &existing.Date, &existing.Payee, &existing.AmountCents, &existing.Currency, &existing.Memo, &existing.ArchiveFilePath, &existing.IsReviewed)
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

// GetUnreviewedTransactions fetches transactions that have not been reviewed.
func (db *DB) GetUnreviewedTransactions(ctx context.Context) ([]model.Transaction, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.date, t.payee, t.amount_cents, t.currency, t.memo, t.archive_file_path, t.category_id, t.is_reviewed, c.name
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.is_reviewed = 0
		ORDER BY t.date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var txs []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.Date, &t.Payee, &t.AmountCents, &t.Currency, &t.Memo, &t.ArchiveFilePath, &t.CategoryID, &t.IsReviewed, &t.CategoryName); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return db.hydrateTags(ctx, txs)
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

// EnrichTransaction updates a transaction with a category, tags, and memo.
// It creates the category and tags if they do not exist.
func (db *DB) EnrichTransaction(ctx context.Context, txID string, categoryName string, tagNames []string, memo string, isReviewed bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var categoryID *string
	if categoryName != "" {
		// Insert category if it doesn't exist (assuming UUID or we can just use the name as ID to be simple).
		// Wait, the migration specifies `id TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL`.
		// Let's use the name as the ID to avoid needing a UUID generator, or if we use UUID, we'd need google/uuid.
		// Since we don't know if google/uuid is used, using the name as the ID is the simplest for categories.
		// Let's check model.go to see if ID is defined. Actually, just using Name as ID for now, or generating one.
		// Let's use name as ID for simplicity and deduplication, but we'll do lowercase or something.
		// Actually, let's just use the name as the ID directly.
		catID := categoryName
		_, err := tx.ExecContext(ctx, "INSERT INTO categories (id, name) VALUES (?, ?) ON CONFLICT(name) DO NOTHING", catID, categoryName)
		if err != nil {
			return fmt.Errorf("failed to upsert category: %w", err)
		}

		err = tx.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = ?", categoryName).Scan(&catID)
		if err != nil {
			return fmt.Errorf("failed to fetch category id: %w", err)
		}
		categoryID = &catID
	}

	// Update the transaction
	_, err = tx.ExecContext(ctx, "UPDATE transactions SET category_id = ?, memo = ?, is_reviewed = ? WHERE id = ?", categoryID, memo, isReviewed, txID)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	// Clear existing tags for the transaction
	_, err = tx.ExecContext(ctx, "DELETE FROM transaction_tags WHERE transaction_id = ?", txID)
	if err != nil {
		return fmt.Errorf("failed to clear transaction tags: %w", err)
	}

	// Upsert and link tags
	for _, tagName := range tagNames {
		if tagName == "" {
			continue
		}
		tagID := tagName
		_, err := tx.ExecContext(ctx, "INSERT INTO tags (id, name) VALUES (?, ?) ON CONFLICT(name) DO NOTHING", tagID, tagName)
		if err != nil {
			return fmt.Errorf("failed to upsert tag %q: %w", tagName, err)
		}

		err = tx.QueryRowContext(ctx, "SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
		if err != nil {
			return fmt.Errorf("failed to fetch tag id for %q: %w", tagName, err)
		}

		_, err = tx.ExecContext(ctx, "INSERT INTO transaction_tags (transaction_id, tag_id) VALUES (?, ?)", txID, tagID)
		if err != nil {
			return fmt.Errorf("failed to link tag %q: %w", tagName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}
	return nil
}

// GetAllTransactions fetches all transactions, both categorized and uncategorized.
func (db *DB) GetAllTransactions(ctx context.Context) ([]model.Transaction, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.date, t.payee, t.amount_cents, t.currency, t.memo, t.archive_file_path, t.category_id, t.is_reviewed, c.name
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
		if err := rows.Scan(&t.ID, &t.Date, &t.Payee, &t.AmountCents, &t.Currency, &t.Memo, &t.ArchiveFilePath, &t.CategoryID, &t.IsReviewed, &t.CategoryName); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return db.hydrateTags(ctx, txs)
}

// GetCategorizedTransactions fetches transactions that have a category assigned.
func (db *DB) GetCategorizedTransactions(ctx context.Context) ([]model.Transaction, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.date, t.payee, t.amount_cents, t.currency, t.memo, t.archive_file_path, t.category_id, t.is_reviewed, c.name
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		ORDER BY t.date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var txs []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.Date, &t.Payee, &t.AmountCents, &t.Currency, &t.Memo, &t.ArchiveFilePath, &t.CategoryID, &t.IsReviewed, &t.CategoryName); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return db.hydrateTags(ctx, txs)
}

func (db *DB) hydrateTags(ctx context.Context, txs []model.Transaction) ([]model.Transaction, error) {
	if len(txs) == 0 {
		return txs, nil
	}

	txIdxMap := make(map[string]int)
	for i, tx := range txs {
		txIdxMap[tx.ID] = i
	}

	rows, err := db.QueryContext(ctx, `
		SELECT tt.transaction_id, t.id, t.name
		FROM transaction_tags tt
		JOIN tags t ON tt.tag_id = t.id
	`)
	if err != nil {
		return nil, fmt.Errorf("query tags failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var txID, tagID, tagName string
		if err := rows.Scan(&txID, &tagID, &tagName); err != nil {
			return nil, fmt.Errorf("scan tag failed: %w", err)
		}

		if idx, exists := txIdxMap[txID]; exists {
			txs[idx].Tags = append(txs[idx].Tags, model.Tag{ID: tagID, Name: tagName})
		}
	}

	return txs, rows.Err()
}

// DeleteCategories removes the specified categories from the database.
func (db *DB) DeleteCategories(ctx context.Context, names []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "DELETE FROM categories WHERE name = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, name := range names {
		if _, err := stmt.ExecContext(ctx, name); err != nil {
			return fmt.Errorf("failed to delete category %q: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}
