package workspace_test

import (
	"path/filepath"
	"testing"

	"github.com/Cakem1x/fin_man/internal/workspace"
)

func TestManagerDBPath(t *testing.T) {
	wsDir := t.TempDir()
	m := workspace.NewManager(wsDir)

	want := filepath.Join(wsDir, "store", "finance.db")
	if got := m.DBPath(); got != want {
		t.Errorf("DBPath() = %q, want %q", got, want)
	}
}
