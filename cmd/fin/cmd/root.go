package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var workDir string

var rootCmd = &cobra.Command{
	Use:   "fin",
	Short: "fin is a CLI for personal finance management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve workdir: %w", err)
		}
		workDir = absWorkDir
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&workDir, "workdir", "C", ".", "Run as if fin was started in <path> instead of the current working directory")
}
