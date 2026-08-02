package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Cakem1x/fin_man/internal/model"
)

func TestDBOpenAndMigrate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fin_man_db_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("failed to remove temp dir: %v", err)
		}
	}()

	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			t.Errorf("failed to close database: %v", err)
		}
	}()

	// Initially, NeedsUpgrade should be true
	needsUpgrade, err := NeedsUpgrade(database.DB)
	if err != nil {
		t.Fatalf("NeedsUpgrade returned error: %v", err)
	}
	if !needsUpgrade {
		t.Errorf("expected NeedsUpgrade to be true on new database, got false")
	}

	// Run migration
	if err := Migrate(database.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// After migration, NeedsUpgrade should be false
	needsUpgrade, err = NeedsUpgrade(database.DB)
	if err != nil {
		t.Fatalf("NeedsUpgrade post-migrate returned error: %v", err)
	}
	if needsUpgrade {
		t.Errorf("expected NeedsUpgrade to be false after migration, got true")
	}

	// Verify table existence by querying sqlite_master
	var tableName string
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='transactions';").Scan(&tableName)
	if err != nil {
		t.Fatalf("failed to find 'transactions' table: %v", err)
	}
	if tableName != "transactions" {
		t.Errorf("expected table name 'transactions', got '%s'", tableName)
	}
}

func TestInsertTransactions_Duplicates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fin_man_db_test_insert_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = database.Close() }()
	if err := Migrate(database.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	date1 := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	txs := []model.Transaction{
		{
			ID:          "hash1",
			Date:        date1,
			Payee:       "Coffee Shop",
			AmountCents: 450,
			Currency:    "EUR",
			Memo:        "Morning coffee",
		},
		{
			ID:          "hash2",
			Date:        date1,
			Payee:       "Bakery",
			AmountCents: 320,
			Currency:    "EUR",
			Memo:        "Croissant",
		},
	}

	ctx := context.Background()
	inserted, duplicates, err := database.InsertTransactions(ctx, txs)
	if err != nil {
		t.Fatalf("unexpected error on first insert: %v", err)
	}
	if inserted != 2 {
		t.Errorf("expected 2 inserts, got %d", inserted)
	}
	if len(duplicates) > 0 {
		t.Errorf("expected no duplicates, got %d", len(duplicates))
	}

	// Insert duplicate for hash1, and a new one (hash3)
	newTxs := []model.Transaction{
		{
			ID:          "hash1",
			Date:        date1,
			Payee:       "Coffee Shop (Changed)",
			AmountCents: 450,
			Currency:    "EUR",
			Memo:        "Morning coffee",
		},
		{
			ID:          "hash3",
			Date:        date1,
			Payee:       "Supermarket",
			AmountCents: 1550,
			Currency:    "EUR",
			Memo:        "Groceries",
		},
	}
	inserted, duplicates, err = database.InsertTransactions(ctx, newTxs)
	if err == nil {
		t.Errorf("expected error due to duplicates, got nil")
	}
	if inserted != 1 {
		t.Errorf("expected 1 insert, got %d", inserted)
	}
	if len(duplicates) != 1 {
		t.Errorf("expected 1 duplicate, got %d", len(duplicates))
	} else {
		pair := duplicates[0]
		if pair.Existing.ID != "hash1" || pair.New.ID != "hash1" {
			t.Errorf("expected duplicate ID hash1, got existing %s, new %s", pair.Existing.ID, pair.New.ID)
		}
		if pair.Existing.Payee != "Coffee Shop" {
			t.Errorf("expected existing payee Coffee Shop, got %s", pair.Existing.Payee)
		}
		if pair.New.Payee != "Coffee Shop (Changed)" {
			t.Errorf("expected new payee Coffee Shop (Changed), got %s", pair.New.Payee)
		}
	}
}
