package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Cakem1x/fin_man/internal/db"
	"github.com/Cakem1x/fin_man/internal/model"
	"github.com/Cakem1x/fin_man/internal/tui"
	"github.com/Cakem1x/fin_man/internal/workspace"
	"github.com/spf13/cobra"
)

var reviewAll bool

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review and enrich uncategorized transactions",
	Long:  `Provides a human enrichment workflow for uncategorized transactions using a terminal UI.`,
	Run: func(cmd *cobra.Command, args []string) {
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
			fmt.Println("Workspace must be opened before reviewing transactions.")
			os.Exit(1)
		}

		dbPath := filepath.Join(wsDir, "store", "finance.db")
		sqliteDB, err := db.Open(dbPath)
		if err != nil {
			log.Fatalf("failed to open database: %v", err)
		}
		defer func() {
			if err := sqliteDB.Close(); err != nil {
				log.Printf("failed to close database: %v", err)
			}
		}()

		ctx := context.Background()

		cats, err := sqliteDB.GetAllCategories(ctx)
		if err != nil {
			log.Fatalf("failed to fetch categories: %v", err)
		}
		catNames := make([]string, len(cats))
		for i, c := range cats {
			catNames[i] = c.Name
		}

		tags, err := sqliteDB.GetAllTags(ctx)
		if err != nil {
			log.Fatalf("failed to fetch tags: %v", err)
		}
		tagNames := make([]string, len(tags))
		for i, t := range tags {
			tagNames[i] = t.Name
		}

		var txs []model.Transaction
		var errTxs error
		if reviewAll {
			txs, errTxs = sqliteDB.GetAllTransactions(ctx)
		} else {
			txs, errTxs = sqliteDB.GetUncategorizedTransactions(ctx)
		}

		if errTxs != nil {
			log.Fatalf("failed to fetch transactions: %v", errTxs)
		}

		if len(txs) == 0 {
			fmt.Println("No transactions found! 🎉")
			return
		}

		for _, tx := range txs {
			res, err := tui.ReviewTransaction(tx, catNames, tagNames)
			if err != nil {
				// User cancelled or error occurred
				fmt.Printf("\nReview aborted: %v\n", err)
				break
			}

			if res.Skip {
				fmt.Printf("Skipped transaction %s\n", tx.ID)
				continue
			}

			// TODO: Save the result to the DB (categories, tags, memo, rules)
			fmt.Printf("Categorized as %s with tags %v. (Save to DB not implemented yet)\n", res.Category, res.Tags)
		}
	},
}

func init() {
	reviewCmd.Flags().BoolVarP(&reviewAll, "all", "a", false, "Review all transactions, including already categorized ones")
	rootCmd.AddCommand(reviewCmd)
}
