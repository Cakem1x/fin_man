package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Cakem1x/fin_man/internal/model"
)

func countRows(t *testing.T, database *DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := database.QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	return n
}

func TestForeignKeys_DeleteTagRemovesTransactionTags(t *testing.T) {
	database := openTestDB(t)
	seedListFixture(t, database)

	if _, err := database.Exec("DELETE FROM tags WHERE name = ?", "daily"); err != nil {
		t.Fatalf("delete tag failed: %v", err)
	}
	if n := countRows(t, database, "SELECT COUNT(*) FROM transaction_tags WHERE tag_id = ?", "daily"); n != 0 {
		t.Errorf("transaction_tags rows for deleted tag = %d, want 0", n)
	}
}

func TestForeignKeys_DeleteCategoryUnsetsTransactionCategory(t *testing.T) {
	database := openTestDB(t)
	seedListFixture(t, database)

	if _, err := database.Exec("DELETE FROM categories WHERE name = ?", "Housing"); err != nil {
		t.Fatalf("delete category failed: %v", err)
	}
	var categoryID sql.NullString
	if err := database.QueryRow("SELECT category_id FROM transactions WHERE id = ?", "rent").Scan(&categoryID); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if categoryID.Valid {
		t.Errorf("category_id = %q, want NULL", categoryID.String)
	}
}

func TestForeignKeys_RejectsTransactionTagWithUnknownTransaction(t *testing.T) {
	database := openTestDB(t)
	seedListFixture(t, database)

	if _, err := database.Exec("INSERT INTO transaction_tags (transaction_id, tag_id) VALUES (?, ?)", "unknown", "daily"); err == nil {
		t.Error("expected insert with unknown transaction ID to fail")
	}
}

func TestForeignKeys_DeleteTransactionRemovesEmbeddings(t *testing.T) {
	database := openTestDB(t)
	seedListFixture(t, database)
	ctx := context.Background()
	if err := database.UpsertEmbeddings(ctx, "m", map[string]model.Embedding{
		"misc": {TextHash: "h", Vector: []float32{1}},
	}); err != nil {
		t.Fatalf("UpsertEmbeddings failed: %v", err)
	}

	if _, err := database.Exec("DELETE FROM transactions WHERE id = ?", "misc"); err != nil {
		t.Fatalf("delete transaction failed: %v", err)
	}
	if n := countRows(t, database, "SELECT COUNT(*) FROM transaction_embeddings WHERE transaction_id = ?", "misc"); n != 0 {
		t.Errorf("transaction_embeddings rows for deleted transaction = %d, want 0", n)
	}
}

func TestForeignKeyViolations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := Migrate(database.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	ctx := context.Background()

	violations, err := database.ForeignKeyViolations(ctx)
	if err != nil {
		t.Fatalf("ForeignKeyViolations failed: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("violations on clean database = %v, want none", violations)
	}

	// A connection without enforcement can write orphaned rows.
	raw, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if _, err := raw.Exec("INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ('gone', 'gone')"); err != nil {
		t.Fatalf("insert orphan failed: %v", err)
	}

	violations, err = database.ForeignKeyViolations(ctx)
	if err != nil {
		t.Fatalf("ForeignKeyViolations failed: %v", err)
	}
	want := []ForeignKeyViolation{
		{Table: "transaction_tags", Parent: "tags", Count: 1},
		{Table: "transaction_tags", Parent: "transactions", Count: 1},
	}
	if !reflect.DeepEqual(violations, want) {
		t.Errorf("violations = %v, want %v", violations, want)
	}
}
