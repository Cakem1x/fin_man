package db

import (
	"context"
	"fmt"
)

// ForeignKeyViolation counts the rows of Table that reference a missing row in Parent.
type ForeignKeyViolation struct {
	Table  string
	Parent string
	Count  int
}

// ForeignKeyViolations runs PRAGMA foreign_key_check and groups the orphaned rows by table and parent table.
func (db *DB) ForeignKeyViolations(ctx context.Context) ([]ForeignKeyViolation, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT "table", parent, COUNT(*)
		FROM pragma_foreign_key_check
		GROUP BY "table", parent
		ORDER BY "table", parent
	`)
	if err != nil {
		return nil, fmt.Errorf("foreign key check failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var violations []ForeignKeyViolation
	for rows.Next() {
		var v ForeignKeyViolation
		if err := rows.Scan(&v.Table, &v.Parent, &v.Count); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		violations = append(violations, v)
	}
	return violations, rows.Err()
}
