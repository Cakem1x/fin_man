package cmd

import (
	"fmt"
	"os"

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

		mgr := workspace.NewManager(workDir)

		if err := mgr.CanInit(); err != nil {
			return fmt.Errorf("init failed: %w", err)
		}

		storePath := resolvePath(storePath)
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
		fmt.Printf("Workspace configured at %s\n", mgr.WorkspaceDir)

		if err := mgr.Open(pwd); err != nil {
			return fmt.Errorf("failed to open workspace: %w", err)
		}
		fmt.Printf("Workspace mounted at %s/store\n", mgr.WorkspaceDir)

		dbExists := true
		if _, err := os.Stat(mgr.DBPath()); os.IsNotExist(err) {
			dbExists = false
		}

		sqliteDB, closeDB, err := openWorkspaceDB(mgr, false)
		if err != nil {
			return err
		}
		defer closeDB()

		if !dbExists {
			if err := db.Migrate(sqliteDB.DB); err != nil {
				return fmt.Errorf("database creation failed: %w", err)
			}
			fmt.Println("New database created successfully.")
		} else {
			if needsUpgrade, _ := db.NeedsUpgrade(sqliteDB.DB); needsUpgrade {
				doMigrate, err := prompt.Confirm("Database is out of date. Trigger migration?")
				if err != nil {
					return fmt.Errorf("prompt failed: %w", err)
				}
				if doMigrate {
					if err := db.Migrate(sqliteDB.DB); err != nil {
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

var workspaceOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the encrypted workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := findWorkspace(workDir)
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
		fmt.Printf("Workspace mounted at %s/store\n", mgr.WorkspaceDir)

		if sqliteDB, closeDB, err := openWorkspaceDB(mgr, false); err == nil {
			if needsUpgrade, _ := db.NeedsUpgrade(sqliteDB.DB); needsUpgrade {
				fmt.Println("\nNotice: Your workspace database is outdated!")
				fmt.Println("Please run 'fin workspace upgrade' to update it.")
			}
			closeDB()
		}
		return nil
	},
}

var workspaceCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Close the encrypted workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := findWorkspace(workDir)
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
		fmt.Printf("Workspace %s/store closed successfully.\n", mgr.WorkspaceDir)
		return nil
	},
}

var workspaceUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the workspace database",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := findWorkspace(workDir)
		if err != nil {
			return err
		}
		sqliteDB, closeDB, err := openWorkspaceDB(mgr, false)
		if err != nil {
			return err
		}
		defer closeDB()
		if err := db.Migrate(sqliteDB.DB); err != nil {
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
