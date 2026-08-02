package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review and enrich uncategorized transactions",
	Long:  `Provides a human enrichment workflow for uncategorized transactions using a terminal UI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("WIP: Starting transaction review TUI...")
		// TODO: Fetch uncategorized transactions from workspace
		// Loop through transactions and call tui.ReviewTransaction
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}
