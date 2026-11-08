package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Cakem1x/fin_man/internal/model"
)

// Filter narrows the result of ListTransactions. Zero values do not filter.
type Filter struct {
	// From is the inclusive lower bound of the transaction date.
	From time.Time
	// To is the exclusive upper bound of the transaction date.
	To time.Time
	// Category matches a category path and all nested categories.
	Category             string
	Reviewed             *bool
	PayeeContains        string
	ExcludeUncategorized bool
}

// categoryPathCondition matches column against path and all nested paths.
// It compares case-sensitively and treats LIKE wildcards in path literally.
func categoryPathCondition(column, path string) (string, []any) {
	cond := fmt.Sprintf("(%[1]s = ? OR substr(%[1]s, 1, length(?) + 1) = ? || '/')", column)
	return cond, []any{path, path, path}
}

func (f Filter) where() (string, []any) {
	conds := []string{"NOT EXISTS (SELECT 1 FROM transactions child WHERE child.parent_id = t.id)"}
	var args []any
	if !f.From.IsZero() {
		conds = append(conds, "t.date >= ?")
		args = append(args, f.From)
	}
	if !f.To.IsZero() {
		conds = append(conds, "t.date < ?")
		args = append(args, f.To)
	}
	if f.Category != "" {
		cond, condArgs := categoryPathCondition("c.name", f.Category)
		conds = append(conds, cond)
		args = append(args, condArgs...)
	}
	if f.Reviewed != nil {
		conds = append(conds, "t.is_reviewed = ?")
		args = append(args, *f.Reviewed)
	}
	if f.PayeeContains != "" {
		conds = append(conds, "instr(lower(t.payee), lower(?)) > 0")
		args = append(args, f.PayeeContains)
	}
	if f.ExcludeUncategorized {
		conds = append(conds, "c.id IS NOT NULL")
	}
	return strings.Join(conds, " AND "), args
}

// ListTransactions returns the transactions that match f, ordered by date.
// It never returns split parents.
func (db *DB) ListTransactions(ctx context.Context, f Filter) ([]model.Transaction, error) {
	where, args := f.where()
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.date, t.payee, t.amount_cents, t.currency, t.memo, t.archive_file_path, t.category_id, t.is_reviewed, t.parent_id, c.name
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE `+where+`
		ORDER BY t.date ASC, t.id ASC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var txs []model.Transaction
	ids := []string{}
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(txs) == 0 {
		return txs, nil
	}

	tags, err := db.loadTags(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range txs {
		txs[i].Tags = tags[txs[i].ID]
	}
	return txs, nil
}

func scanTransaction(rows *sql.Rows) (model.Transaction, error) {
	var t model.Transaction
	err := rows.Scan(&t.ID, &t.Date, &t.Payee, &t.AmountCents, &t.Currency, &t.Memo, &t.ArchiveFilePath, &t.CategoryID, &t.IsReviewed, &t.ParentID, &t.CategoryName)
	if err != nil {
		return t, fmt.Errorf("scan failed: %w", err)
	}
	return t, nil
}

// loadTags returns the tags of the given transactions, keyed by transaction ID.
func (db *DB) loadTags(ctx context.Context, txIDs []string) (map[string][]model.Tag, error) {
	// A single JSON array parameter avoids SQLite's limit on bound parameters.
	idsJSON, err := json.Marshal(txIDs)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT tt.transaction_id, t.id, t.name
		FROM transaction_tags tt
		JOIN tags t ON tt.tag_id = t.id
		WHERE tt.transaction_id IN (SELECT value FROM json_each(?))
		ORDER BY t.name ASC
	`, string(idsJSON))
	if err != nil {
		return nil, fmt.Errorf("query tags failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tags := make(map[string][]model.Tag)
	for rows.Next() {
		var txID string
		var tag model.Tag
		if err := rows.Scan(&txID, &tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("scan tag failed: %w", err)
		}
		tags[txID] = append(tags[txID], tag)
	}
	return tags, rows.Err()
}
