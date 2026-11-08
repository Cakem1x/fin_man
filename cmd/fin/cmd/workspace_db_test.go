package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cakem1x/fin_man/internal/db"
)

// newClosedWorkspace creates a workspace whose store/ is a plain directory.
// IsOpen reports such a workspace as closed.
func newClosedWorkspace(t *testing.T) string {
	t.Helper()
	wsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(wsDir, "fin.toml"), []byte("store_path = \"cipher\"\n"), 0644); err != nil {
		t.Fatalf("failed to write fin.toml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(wsDir, "store"), 0755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	return wsDir
}

func TestOpenWorkspaceDB_ClosedWorkspace(t *testing.T) {
	wsDir := newClosedWorkspace(t)
	mgr, err := findWorkspace(wsDir)
	if err != nil {
		t.Fatalf("findWorkspace failed: %v", err)
	}

	_, _, err = openWorkspaceDB(mgr, true)
	if !errors.Is(err, errWorkspaceClosed) {
		t.Fatalf("expected errWorkspaceClosed, got %v", err)
	}
}

func TestFindWorkspace_FromSubdirectory(t *testing.T) {
	wsDir := newClosedWorkspace(t)
	subDir := filepath.Join(wsDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	mgr, err := findWorkspace(subDir)
	if err != nil {
		t.Fatalf("findWorkspace failed: %v", err)
	}
	if mgr.WorkspaceDir != wsDir {
		t.Errorf("WorkspaceDir = %q, want %q", mgr.WorkspaceDir, wsDir)
	}
}

func TestOpenDB_OutdatedSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "finance.db")

	_, _, err := openDB(dbPath, true)
	if !errors.Is(err, errSchemaOutdated) {
		t.Fatalf("expected errSchemaOutdated, got %v", err)
	}
	if !strings.Contains(err.Error(), "fin workspace upgrade") {
		t.Errorf("expected error to mention 'fin workspace upgrade', got %q", err.Error())
	}
}

func TestOpenDB_OutdatedSchemaAllowed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "finance.db")

	database, closeDB, err := openDB(dbPath, false)
	if err != nil {
		t.Fatalf("openDB failed: %v", err)
	}
	defer closeDB()
	if database == nil {
		t.Fatal("expected a database")
	}
}

func TestOpenDB_UpToDateSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "finance.db")
	setup, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	if err := db.Migrate(setup.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	if err := setup.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	database, closeDB, err := openDB(dbPath, true)
	if err != nil {
		t.Fatalf("openDB failed: %v", err)
	}
	defer closeDB()
	if database == nil {
		t.Fatal("expected a database")
	}
}
