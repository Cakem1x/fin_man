package db

import (
	"os"
	"path/filepath"
	"testing"
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
