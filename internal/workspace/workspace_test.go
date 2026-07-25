package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Cakem1x/fin_man/internal/workspace"
)

func TestWorkspaceLifecycle(t *testing.T) {
	wsDir := t.TempDir()
	cipherDir := filepath.Join(t.TempDir(), "finance.cipher") // Store somewhere outside to test linking

	m := workspace.NewManager(wsDir)

	t.Run("InitNewStore", func(t *testing.T) {
		err := m.Init(cipherDir, "test-password")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Check that cipher dir exists and has gocryptfs.conf
		if _, err := os.Stat(filepath.Join(cipherDir, "gocryptfs.conf")); os.IsNotExist(err) {
			t.Errorf("gocryptfs.conf was not created")
		}

		// Check that fin.toml was created
		if _, err := os.Stat(filepath.Join(wsDir, "fin.toml")); os.IsNotExist(err) {
			t.Errorf("fin.toml was not created")
		}
	})

	t.Run("InitExistingStore", func(t *testing.T) {
		// Create a second workspace dir to link to the same cipherDir
		secondWsDir := t.TempDir()
		m2 := workspace.NewManager(secondWsDir)

		err := m2.Init(cipherDir, "test-password")
		if err != nil {
			t.Fatalf("Init existing failed: %v", err)
		}

		// Check that fin.toml was created in the second workspace
		if _, err := os.Stat(filepath.Join(secondWsDir, "fin.toml")); os.IsNotExist(err) {
			t.Errorf("fin.toml was not created in second workspace")
		}
	})

	t.Run("Open", func(t *testing.T) {
		err := m.Open("test-password")
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}

		// Prevent Double Open Check
		isOpen, _ := m.IsOpen()
		if !isOpen {
			t.Fatalf("Expected workspace to be open")
		}

		err = m.Open("test-password")
		if err == nil {
			t.Fatalf("Expected double open to fail, but it succeeded")
		}

		mountDir := filepath.Join(wsDir, "store")
		// Ensure the mount directory was created and is accessible
		if _, err := os.Stat(mountDir); os.IsNotExist(err) {
			t.Errorf("store dir does not exist")
		}

		// Create a file in the mounted workspace to verify it works
		testFile := filepath.Join(mountDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to write to mounted workspace: %v", err)
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := m.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		mountDir := filepath.Join(wsDir, "store")
		testFile := filepath.Join(mountDir, "test.txt")
		if _, err := os.Stat(testFile); err == nil {
			t.Errorf("file still visible after close, meaning unmount failed")
		}

		// Prevent Double Close Check
		isOpen, _ := m.IsOpen()
		if isOpen {
			t.Fatalf("Expected workspace to be closed")
		}

		err = m.Close()
		if err == nil {
			t.Fatalf("Expected double close to fail, but it succeeded")
		}
	})

	t.Run("PersistenceAndRecovery", func(t *testing.T) {
		// Delete the entire workspace directory, but the cipherDir (store) remains untouched
		if err := os.RemoveAll(wsDir); err != nil {
			t.Fatalf("Failed to remove wsDir: %v", err)
		}

		// Create a brand new workspace directory
		newWsDir := t.TempDir()
		mNew := workspace.NewManager(newWsDir)

		// Init from existing store (we shouldn't need a password here)
		exists, _ := mNew.IsStoreInitialized(cipherDir)
		if !exists {
			t.Fatalf("Expected store to exist")
		}
		if err := mNew.Init(cipherDir, ""); err != nil {
			t.Fatalf("Init existing failed: %v", err)
		}

		// Open it with the original password
		if err := mNew.Open("test-password"); err != nil {
			t.Fatalf("Open existing failed: %v", err)
		}

		// Verify the data persisted
		newMountDir := filepath.Join(newWsDir, "store")
		testFile := filepath.Join(newMountDir, "test.txt")
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read persisted file: %v", err)
		}
		if string(content) != "hello" {
			t.Errorf("Expected 'hello', got '%s'", string(content))
		}

		if err := mNew.Close(); err != nil {
			t.Fatalf("Failed to close workspace: %v", err)
		}
	})

	t.Run("PreventNestedWorkspace", func(t *testing.T) {
		parentWs := t.TempDir()
		mParent := workspace.NewManager(parentWs)
		if err := mParent.Init(cipherDir, "test-password"); err != nil {
			t.Fatalf("Failed to init parent workspace: %v", err)
		}

		// Create a subdirectory inside the already initialized parentWs
		nestedDir := filepath.Join(parentWs, "subfolder")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatalf("Failed to create nested dir: %v", err)
		}

		// Attempt to init a new workspace inside the subfolder
		mNested := workspace.NewManager(nestedDir)
		err := mNested.Init(cipherDir, "test-password")
		if err == nil {
			t.Fatalf("Expected Init to fail on nested workspace, but it succeeded")
		}
	})
}
