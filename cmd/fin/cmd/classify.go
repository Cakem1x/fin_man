package cmd

import (
	"context"
	"fmt"

	"github.com/Cakem1x/fin_man/internal/categorize"
	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/ollama"
	"github.com/Cakem1x/fin_man/internal/workspace"
	"github.com/spf13/cobra"
)

var classifyCmd = &cobra.Command{
	Use:   "classify",
	Short: "Compute missing transaction embeddings with Ollama",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := findWorkspace(workDir)
		if err != nil {
			return err
		}
		cfg, err := mgr.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to read fin.toml: %w", err)
		}
		if !cfg.LLM.Enabled {
			return fmt.Errorf("LLM support is disabled, set 'enabled = true' in the [llm] section of fin.toml")
		}

		dbConn, closeDB, err := openWorkspaceDB(mgr, true)
		if err != nil {
			return err
		}
		defer closeDB()

		if err := embedTransactions(context.Background(), cfg.LLM, dbConn); err != nil {
			return fmt.Errorf("embedding failed: %w", err)
		}
		return nil
	},
}

// embedTransactions embeds all transactions without up-to-date embedding and
// prints the progress.
func embedTransactions(ctx context.Context, llm workspace.LLMConfig, dbConn *db.DB) error {
	txs, err := dbConn.ListTransactions(ctx, db.Filter{})
	if err != nil {
		return fmt.Errorf("failed to fetch transactions: %w", err)
	}
	embedded := 0
	_, err = categorize.EnsureEmbeddings(ctx, dbConn, ollama.NewClient(llm.Endpoint), llm.EmbedModel, txs,
		func(done, total int) {
			embedded = done
			fmt.Printf("\rEmbedded %d/%d transactions", done, total)
		})
	if embedded > 0 {
		fmt.Println()
	} else if err == nil {
		fmt.Println("All transactions have up-to-date embeddings")
	}
	return err
}

func init() {
	rootCmd.AddCommand(classifyCmd)
}
