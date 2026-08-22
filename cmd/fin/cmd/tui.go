package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/tui"
	"github.com/Cakem1x/fin_man/internal/workspace"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the TUI to explore your categorized finances",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cwd, _ := os.Getwd()
		wsDir, err := workspace.FindRoot(cwd)
		if err != nil {
			log.Fatalf("failed to find workspace: %v", err)
		}

		mgr := workspace.NewManager(wsDir)
		isOpen, err := mgr.IsOpen()
		if err != nil {
			log.Fatalf("failed to check workspace state: %v", err)
		}
		if !isOpen {
			fmt.Println("Workspace must be opened before viewing tui.")
			os.Exit(1)
		}

		dbPath := filepath.Join(wsDir, "store", "finance.db")
		dbConn, err := db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			if err := dbConn.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()

		// Fetch all transactions to allow memory-based dynamic filtering
		txs, err := dbConn.GetAllTransactions(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch transactions: %w", err)
		}

		m := tui.NewOverviewModel(txs, dbConn)
		p := tea.NewProgram(&m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
