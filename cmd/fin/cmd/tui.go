package cmd

import (
	"context"
	"fmt"

	"github.com/Cakem1x/fin_man/internal/app"
	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/db"
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

		var embeddings map[string][]float32
		cfg, err := mgr.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to read fin.toml: %w", err)
		}
		if cfg.LLM.Enabled {
			txs, err := dbConn.ListTransactions(ctx, db.Filter{})
			if err != nil {
				return fmt.Errorf("failed to fetch transactions: %w", err)
			}
			embeddings, err = categorize.LoadEmbeddings(ctx, dbConn, cfg.LLM.EmbedModel, txs)
			if err != nil {
				return fmt.Errorf("failed to load embeddings: %w", err)
			}
		}

		svc, err := app.New(ctx, dbConn, embeddings)
		if err != nil {
			return err
		}
		m := tui.NewOverviewModel(svc)
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
