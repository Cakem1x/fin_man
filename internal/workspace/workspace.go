package workspace

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	StorePath string `toml:"store_path"`
}

// Manager handles the lifecycle of a git-like encrypted finance workspace.
type Manager struct {
	WorkspaceDir string
}

// FindRoot walks up the directory tree to find the workspace root (where fin.toml is).
func FindRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "fin.toml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a fin workspace (or any of the parent directories): no fin.toml found")
		}
		dir = parent
	}
}

func NewManager(workspaceDir string) *Manager {
	return &Manager{WorkspaceDir: workspaceDir}
}

func (m *Manager) runGocryptfs(args []string, password string) error {
	f, err := os.CreateTemp("", "gocryptfs-pass-*")
	if err != nil {
		return fmt.Errorf("failed to create passfile: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	if _, err := f.WriteString(password); err != nil {
		return fmt.Errorf("failed to write passfile: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close passfile: %w", err)
	}

	fullArgs := append(args, "-passfile", f.Name())
	cmd := exec.Command("gocryptfs", fullArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gocryptfs failed: %v, stderr: %s", err, stderr.String())
	}
	return nil
}

// IsStoreInitialized checks if the gocryptfs store already exists at the given path.
func (m *Manager) IsStoreInitialized(storePath string) (bool, error) {
	absStorePath, err := filepath.Abs(storePath)
	if err != nil {
		return false, err
	}
	confPath := filepath.Join(absStorePath, "gocryptfs.conf")
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}

// CanInit returns an error if the manager's directory is already inside an initialized workspace.
func (m *Manager) CanInit() error {
	if root, err := FindRoot(m.WorkspaceDir); err == nil {
		return fmt.Errorf("already inside an initialized workspace (root: %s)", root)
	}
	return nil
}

func (m *Manager) Init(storePath, password string) error {
	if err := m.CanInit(); err != nil {
		return err
	}

	if err := os.MkdirAll(m.WorkspaceDir, 0755); err != nil {
		return err
	}

	absStorePath, err := filepath.Abs(storePath)
	if err != nil {
		return err
	}

	confPath := filepath.Join(absStorePath, "gocryptfs.conf")
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absStorePath, 0755); err != nil {
			return fmt.Errorf("failed to create store dir: %w", err)
		}
		if err := m.runGocryptfs([]string{"-init", absStorePath}, password); err != nil {
			return err
		}
	}

	// Make relative to workspace if possible for portability, else absolute
	relStorePath, err := filepath.Rel(m.WorkspaceDir, absStorePath)
	if err != nil {
		relStorePath = absStorePath
	}

	cfg := Config{StorePath: relStorePath}
	b, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(m.WorkspaceDir, "fin.toml"), b, 0644); err != nil {
		return err
	}

	return nil
}

func (m *Manager) loadConfig() (*Config, error) {
	b, err := os.ReadFile(filepath.Join(m.WorkspaceDir, "fin.toml"))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// IsOpen checks if the store directory is currently mounted as a FUSE filesystem.
func (m *Manager) IsOpen() (bool, error) {
	mountDir := filepath.Join(m.WorkspaceDir, "store")
	fi, err := os.Lstat(mountDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	parentFi, err := os.Lstat(m.WorkspaceDir)
	if err != nil {
		return false, err
	}

	stat, ok1 := fi.Sys().(*syscall.Stat_t)
	parentStat, ok2 := parentFi.Sys().(*syscall.Stat_t)
	if ok1 && ok2 {
		return stat.Dev != parentStat.Dev, nil
	}
	return false, nil
}

func (m *Manager) Open(password string) error {
	isOpen, err := m.IsOpen()
	if err != nil {
		return err
	}
	if isOpen {
		return fmt.Errorf("workspace is already open")
	}

	cfg, err := m.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to read fin.toml: %w", err)
	}

	storePath := cfg.StorePath
	if !filepath.IsAbs(storePath) {
		storePath = filepath.Join(m.WorkspaceDir, storePath)
	}

	mountDir := filepath.Join(m.WorkspaceDir, "store")
	if err := os.MkdirAll(mountDir, 0755); err != nil {
		return err
	}

	if err := m.runGocryptfs([]string{storePath, mountDir}, password); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Close() error {
	isOpen, err := m.IsOpen()
	if err != nil {
		return err
	}
	if !isOpen {
		return fmt.Errorf("workspace is already closed")
	}

	mountDir := filepath.Join(m.WorkspaceDir, "store")
	cmd := exec.Command("fusermount", "-u", mountDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fusermount failed: %v, stderr: %s", err, stderr.String())
	}
	return nil
}
