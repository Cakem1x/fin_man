package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/prompt"
	"github.com/Cakem1x/fin_man/internal/workspace"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage encrypted workspace (init, open, close, upgrade)",
}

var storePath string

var workspaceInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		if storePath == "" {
			return fmt.Errorf("--store flag is required for init")
		}

		cwd, _ := os.Getwd()
		mgr := workspace.NewManager(cwd)

		if err := mgr.CanInit(); err != nil {
			return fmt.Errorf("init failed: %w", err)
		}

		exists, err := mgr.IsStoreInitialized(storePath)
		if err != nil {
			return fmt.Errorf("failed to check store: %w", err)
		}

		var pwd string
		if exists {
			fmt.Printf("Initializing workspace from existing store at %s\n", storePath)
			pwd, err = prompt.ReadPassword("Enter workspace password")
			if err != nil {
				return fmt.Errorf("password prompt failed: %w", err)
			}
		} else {
			fmt.Printf("Initializing workspace and creating new store at %s\n", storePath)
			pwd, err = prompt.ReadNewPassword("Enter new workspace password", "Confirm new workspace password")
			if err != nil {
				return fmt.Errorf("password prompt failed: %w", err)
			}
		}

		if err := mgr.Init(storePath, pwd); err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		fmt.Printf("Workspace configured at %s\n", cwd)

		if err := mgr.Open(pwd); err != nil {
			return fmt.Errorf("failed to open workspace: %w", err)
		}
		fmt.Printf("Workspace mounted at %s/store\n", cwd)

		dbPath := filepath.Join(cwd, "store", "finance.db")
		dbExists := true
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			dbExists = false
		}

		sqliteDB, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			if err := sqliteDB.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()

		if !dbExists {
			if err := db.Migrate(sqliteDB); err != nil {
				return fmt.Errorf("database creation failed: %w", err)
			}
			fmt.Println("New database created successfully.")
		} else {
			if needsUpgrade, _ := db.NeedsUpgrade(sqliteDB); needsUpgrade {
				doMigrate, err := prompt.Confirm("Database is out of date. Trigger migration?")
				if err != nil {
					return fmt.Errorf("prompt failed: %w", err)
				}
				if doMigrate {
					if err := db.Migrate(sqliteDB); err != nil {
						return fmt.Errorf("migration failed: %w", err)
					}
					fmt.Println("Workspace database upgraded successfully.")
				} else {
					fmt.Println("Notice: Workspace database is out of date and not usable before running 'fin workspace upgrade'")
				}
			}
		}
		return nil
	},
}

func getWorkspaceManager() (*workspace.Manager, string, error) {
	cwd, _ := os.Getwd()
	wsDir, err := workspace.FindRoot(cwd)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find workspace root: %w", err)
	}
	return workspace.NewManager(wsDir), wsDir, nil
}

var workspaceOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the encrypted workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, wsDir, err := getWorkspaceManager()
		if err != nil {
			return err
		}
		isOpen, err := mgr.IsOpen()
		if err != nil {
			return fmt.Errorf("failed to check workspace state: %w", err)
		}
		if isOpen {
			fmt.Println("Workspace is already open.")
			return nil
		}

		pwd, err := prompt.ReadPassword("Enter workspace password")
		if err != nil {
			return fmt.Errorf("password prompt failed: %w", err)
		}
		if err := mgr.Open(pwd); err != nil {
			return fmt.Errorf("open failed: %w", err)
		}
		cwd, _ := os.Getwd()
		fmt.Printf("Workspace mounted at %s/store\n", cwd)

		dbPath := filepath.Join(wsDir, "store", "finance.db")
		if sqliteDB, err := sql.Open("sqlite3", dbPath); err == nil {
			if needsUpgrade, _ := db.NeedsUpgrade(sqliteDB); needsUpgrade {
				fmt.Println("\nNotice: Your workspace database is outdated!")
				fmt.Println("Please run 'fin workspace upgrade' to update it.")
			}
			if err := sqliteDB.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}
		return nil
	},
}

var workspaceCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Close the encrypted workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, _, err := getWorkspaceManager()
		if err != nil {
			return err
		}
		isOpen, err := mgr.IsOpen()
		if err != nil {
			return fmt.Errorf("failed to check workspace state: %w", err)
		}
		if !isOpen {
			fmt.Println("Workspace is already closed.")
			return nil
		}

		if err := mgr.Close(); err != nil {
			return fmt.Errorf("close failed: %w", err)
		}
		cwd, _ := os.Getwd()
		fmt.Printf("Workspace %s/store closed successfully.\n", cwd)
		return nil
	},
}

var workspaceUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the workspace database",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, wsDir, err := getWorkspaceManager()
		if err != nil {
			return err
		}
		isOpen, err := mgr.IsOpen()
		if err != nil {
			return fmt.Errorf("failed to check workspace state: %w", err)
		}
		if !isOpen {
			return fmt.Errorf("workspace must be opened before upgrading the database")
		}
		dbPath := filepath.Join(wsDir, "store", "finance.db")
		sqliteDB, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			if err := sqliteDB.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()
		if err := db.Migrate(sqliteDB); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println("Workspace database upgraded successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.AddCommand(workspaceInitCmd)
	workspaceCmd.AddCommand(workspaceOpenCmd)
	workspaceCmd.AddCommand(workspaceCloseCmd)
	workspaceCmd.AddCommand(workspaceUpgradeCmd)

	workspaceInitCmd.Flags().StringVar(&storePath, "store", "", "Path to the encrypted store (required for init)")
}
