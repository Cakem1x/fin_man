package cmd

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/workspace"
)

var (
	errWorkspaceClosed = errors.New("workspace is not open, run 'fin workspace open' first")
	errSchemaOutdated  = errors.New("database schema is outdated, run 'fin workspace upgrade' first")
)

// resolvePath interprets a relative path relative to the --workdir directory.
func resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workDir, path)
}

// findWorkspace returns a manager for the workspace that contains startDir.
func findWorkspace(startDir string) (*workspace.Manager, error) {
	wsDir, err := workspace.FindRoot(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find workspace root: %w", err)
	}
	return workspace.NewManager(wsDir), nil
}

// openWorkspaceDB opens the database of an open workspace.
// It returns errWorkspaceClosed if the workspace is closed.
// With requireUpToDate, it returns errSchemaOutdated if migrations are pending.
// The returned function closes the database.
func openWorkspaceDB(mgr *workspace.Manager, requireUpToDate bool) (*db.DB, func(), error) {
	isOpen, err := mgr.IsOpen()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check workspace state: %w", err)
	}
	if !isOpen {
		return nil, nil, errWorkspaceClosed
	}
	return openDB(mgr.DBPath(), requireUpToDate)
}

// openDB opens the database at dbPath.
// With requireUpToDate, it returns errSchemaOutdated if migrations are pending.
// The returned function closes the database.
func openDB(dbPath string, requireUpToDate bool) (*db.DB, func(), error) {
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	closeDB := func() {
		if err := database.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}
	if requireUpToDate {
		needsUpgrade, err := db.NeedsUpgrade(database.DB)
		if err != nil {
			closeDB()
			return nil, nil, fmt.Errorf("failed to check database schema: %w", err)
		}
		if needsUpgrade {
			closeDB()
			return nil, nil, errSchemaOutdated
		}
	}
	return database, closeDB, nil
}
