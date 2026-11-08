package cmd

import (
	"context"
	"fmt"

	"github.com/Cakem1x/fin_man/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the TUI to explore your categorized finances",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		mgr, err := findWorkspace(workDir)
		if err != nil {
			return err
		}
		dbConn, closeDB, err := openWorkspaceDB(mgr, true)
		if err != nil {
			return err
		}
		defer closeDB()

		// Fetch all transactions to allow memory-based dynamic filtering
		txs, err := dbConn.ListTransactions(ctx, db.Filter{})
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
